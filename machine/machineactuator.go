/*
Copyright 2018 Platform 9 Systems, Inc.
*/

package machine

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pkg/sftp"
	"github.com/platform9/ssh-provider/provisionedmachine"

	"golang.org/x/crypto/ssh"

	sshconfigv1 "github.com/platform9/ssh-provider/sshproviderconfig/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterutil "sigs.k8s.io/cluster-api/pkg/util"
)

type SSHActuator struct {
	InsecureIgnoreHostKey bool
	sshProviderCodec      *sshconfigv1.SSHProviderCodec

	provisionedMachineConfigMaps []*corev1.ConfigMap
	sshCredentials               *corev1.Secret
	etcdCA                       *corev1.Secret
	apiServerCA                  *corev1.Secret
	frontProxyCA                 *corev1.Secret
	serviceAccountKey            *corev1.Secret
}

func NewActuator(provisionedMachineConfigMaps []*corev1.ConfigMap,
	sshCredentials *corev1.Secret,
	etcdCA *corev1.Secret,
	apiServerCA *corev1.Secret,
	frontProxyCA *corev1.Secret,
	serviceAccountKey *corev1.Secret) (*SSHActuator, error) {
	codec, err := sshconfigv1.NewCodec()
	if err != nil {
		return nil, err
	}
	return &SSHActuator{
		sshProviderCodec:             codec,
		provisionedMachineConfigMaps: provisionedMachineConfigMaps,
		sshCredentials:               sshCredentials,
		etcdCA:                       etcdCA,
		apiServerCA:                  apiServerCA,
		frontProxyCA:                 frontProxyCA,
		serviceAccountKey:            serviceAccountKey,
	}, nil
}

func (sa *SSHActuator) Create(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	cm, err := sa.ReserveProvisionedMachine(machine)
	if err != nil {
		return fmt.Errorf("error creating machine: error reserving provisioned machine %q: %s", machine.Name, err)
	}

	client, err := sshClient(cm, sa.sshCredentials, sa.InsecureIgnoreHostKey)
	if err != nil {
		return fmt.Errorf("error creating machine %q: failed to create SSH client: %s", machine.Name, err)
	}
	defer client.Close()

	pm, err := provisionedmachine.NewFromConfigMap(cm)
	if err != nil {
		return fmt.Errorf("error creating machine: error parsing ProvisionedMachine from ConfigMap %q: %s", cm.Name, err)
	}
	if clusterutil.IsMaster(machine) {
		if err := sa.createMaster(pm, cluster, machine, client); err != nil {
			return fmt.Errorf("error creating machine %q: %s", machine.Name, err)
		}
	} else {
		if err := sa.createNode(cluster, machine, client); err != nil {
			return fmt.Errorf("error creating machine %q: %s", machine.Name, err)
		}
	}
	return nil
}

func (sa *SSHActuator) createMaster(pm *provisionedmachine.ProvisionedMachine, cluster *clusterv1.Cluster, machine *clusterv1.Machine, client *ssh.Client) error {
	var err error

	nodeadmConfiguration, err := sa.NewNodeadmConfiguration(pm, cluster, machine)
	if err != nil {
		return err
	}

	mcb, err := MarshalToYAMLWithFixedKubeProxyFeatureGates(nodeadmConfiguration)
	if err != nil {
		return err
	}

	sftp, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("error creating SFTP client: %s", err)
	}
	defer sftp.Close()

	f, err := sftp.Create("/tmp/nodeadm.yaml")
	if err != nil {
		return fmt.Errorf("error creating kubeadm.yaml: %s", err)
	}
	if _, err := f.Write(mcb); err != nil {
		return fmt.Errorf("error writing kubeadm.yaml: %s", err)
	}

	var session *ssh.Session
	var out []byte
	session, err = client.NewSession()
	defer session.Close()
	if err != nil {
		return fmt.Errorf("error creating new SSH session for machine %q: %s", machine.Name, err)
	}
	out, err = session.CombinedOutput("echo writing ca cert and key")
	if err != nil {
		return fmt.Errorf("error invoking ssh command %s", err)
	}
	log.Println(string(out))

	session, err = client.NewSession()
	defer session.Close()
	if err != nil {
		return fmt.Errorf("error creating new SSH session for machine %q: %s", machine.Name, err)
	}
	out, err = session.CombinedOutput("/opt/bin/etcdadm init")
	if err != nil {
		return fmt.Errorf("error invoking ssh command %s", err)
	}
	log.Println(string(out))

	session, err = client.NewSession()
	defer session.Close()
	if err != nil {
		return fmt.Errorf("error creating new SSH session for machine %q: %s", machine.Name, err)
	}
	out, err = session.CombinedOutput("/opt/bin/etcdadm info")
	if err != nil {
		return fmt.Errorf("error invoking ssh command %s", err)
	}
	etcdMember := sshconfigv1.EtcdMember{}
	err = json.Unmarshal(out, &etcdMember)
	if err != nil {
		return fmt.Errorf("error reading etcdadm info: %s", err)
	}
	mps := &sshconfigv1.SSHMachineProviderStatus{
		EtcdMember: &etcdMember,
	}
	sa.sshProviderCodec.DecodeFromProviderStatus(machine.Status.ProviderStatus, mps)
	ps, err := sa.sshProviderCodec.EncodeToProviderStatus(mps)
	if err != nil {
		return fmt.Errorf("error encoding machine provider status: %s", err)
	}
	machine.Status.ProviderStatus = *ps

	session, err = client.NewSession()
	defer session.Close()
	if err != nil {
		return fmt.Errorf("error creating new SSH session for machine %q: %s", machine.Name, err)
	}
	out, err = session.CombinedOutput("/opt/bin/nodeadm init --cfg /tmp/nodeadm.yaml")
	if err != nil {
		return fmt.Errorf("error invoking ssh command %s", err)
	}
	log.Println(string(out))

	// TODO(dlipovetsky) Update cluster CA Secret with actual CA

	return nil
}

func (sa *SSHActuator) createNode(cluster *clusterv1.Cluster, machine *clusterv1.Machine, client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("error creating new SSH session for machine %q: %s", machine.Name, err)
	}
	out, err := session.CombinedOutput("echo running nodeadm join")
	if err != nil {
		return fmt.Errorf("error invoking ssh command %s", err)
	}
	log.Println(out)
	return nil
}

func (sa *SSHActuator) Delete(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	return nil
}

func (sa *SSHActuator) Update(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	return nil
}

func (sa *SSHActuator) Exists(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	return false, nil
}

func (sa *SSHActuator) machineproviderconfig(providerConfig clusterv1.ProviderConfig) (*sshconfigv1.SSHMachineProviderConfig, error) {
	var config sshconfigv1.SSHMachineProviderConfig
	err := sa.sshProviderCodec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, fmt.Errorf("error decoding SSHMachineProviderConfig from ProviderConfig: %s", err)
	}
	return &config, nil
}

func (sa *SSHActuator) clusterproviderconfig(providerConfig clusterv1.ProviderConfig) (*sshconfigv1.SSHClusterProviderConfig, error) {
	var config sshconfigv1.SSHClusterProviderConfig
	err := sa.sshProviderCodec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, fmt.Errorf("error decoding SSHClusterProviderConfig from ProviderConfig: %s", err)
	}
	return &config, nil
}
