package sekaid

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/KiraCore/ryokai/internal/core/orchestration/docker"
	"github.com/KiraCore/ryokai/pkg/ryokaicommon/types/constants"
	osutils "github.com/KiraCore/ryokai/pkg/ryokaicommon/utils/os"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/google/shlex"

	vlg "github.com/PeepoFrog/validator-key-gen/MnemonicsGenerator"
	"github.com/miekg/dns"
)

type SekaiPlugin struct {
	dockerOrchestrator *docker.DockerOrchestrator
	sekaidConfig       *SekaidConfig
}

type SekaidConfig struct {
	MasterMnemonicSet   *vlg.MasterMnemonicSet `toml:"-"`
	SecretsFolder       string                 // Path to mnemonics.env and node keys
	Moniker             string                 // Moniker
	SekaidHome          string                 // Home folder for sekai bin
	NetworkName         string                 // Name of a blockchain name (chain-ID)
	SekaidContainerName string                 // Name for sekai container
	KeyringBackend      string                 // Name of keyring backend
	RpcPort             string                 // Sekaid's rpc port
	GrpcPort            string                 // Sekaid's grpc port
	P2PPort             string                 // Sekaid's p2p port
	PrometheusPort      string                 // Prometheus port
	MnemonicDir         string                 // Destination where mnemonics file will be saved
}

func NewSekaiPlugin(ctx context.Context) (*SekaiPlugin, error) {
	dockerOrchestrator, err := docker.NewDockerOrchestrator()
	if err != nil {
		return nil, err
	}

	return &SekaiPlugin{dockerOrchestrator: dockerOrchestrator}, nil
}

func (sekaiPlugin *SekaiPlugin) InitNewSekaid(ctx context.Context) error {
	os.Exit(1)
	// log := logging.Log
	//  log.Infof("Setting up '%s' (sekaid) genesis container", sekaiPlugin.sekaidConfig.SekaidContainerName)
	// Have to do this because need to initialize sekaid folder
	initcmd := fmt.Sprintf(`sekaid init  --overwrite --chain-id=%s --home=%s "%s"`, sekaiPlugin.sekaidConfig.NetworkName, sekaiPlugin.sekaidConfig.SekaidHome, sekaiPlugin.sekaidConfig.Moniker)
	//  log.Tracef("running %s\n", initcmd)
	_, err := sekaiPlugin.dockerOrchestrator.ExecCommandInContainer(ctx, sekaiPlugin.sekaidConfig.SekaidContainerName, initcmd)
	//  log.Tracef("out: %s, err:%v\n", string(out), err)
	err = sekaiPlugin.SetSekaidKeys(ctx)
	if err != nil {
		//  log.Errorf("Can't set sekaid keys: %s", err)
		return fmt.Errorf("can't set sekaid keys %w", err)
	}
	// sekaiPlugin.dockerOrchestrator.
	err = sekaiPlugin.setEmptyValidatorState(ctx)
	if err != nil {
		//  log.Errorf("Setting empty validator state error: %s", err)
		return err
	}

	commands := []string{
		fmt.Sprintf(`sekaid init  --overwrite --chain-id=%s --home=%s "%s"`,
			sekaiPlugin.sekaidConfig.NetworkName, sekaiPlugin.sekaidConfig.SekaidHome, sekaiPlugin.sekaidConfig.Moniker),
		fmt.Sprintf("mkdir %s", sekaiPlugin.sekaidConfig.MnemonicDir),
		fmt.Sprintf(`yes %s | sekaid keys add "%s" --keyring-backend=%s --home=%s --output=json --recover | jq .mnemonic > %s/sekai.mnemonic`,
			sekaiPlugin.sekaidConfig.MasterMnemonicSet.ValidatorAddrMnemonic, constants.ValidatorAccountName, sekaiPlugin.sekaidConfig.KeyringBackend, sekaiPlugin.sekaidConfig.SekaidHome, sekaiPlugin.sekaidConfig.MnemonicDir),
		fmt.Sprintf(`yes %s | sekaid keys add "signer" --keyring-backend=%s --home=%s --output=json --recover | jq .mnemonic > %s/sekai.mnemonic`,
			sekaiPlugin.sekaidConfig.MasterMnemonicSet.SignerAddrMnemonic, sekaiPlugin.sekaidConfig.KeyringBackend, sekaiPlugin.sekaidConfig.SekaidHome, sekaiPlugin.sekaidConfig.MnemonicDir),
		fmt.Sprintf(`sekaid keys add "faucet" --keyring-backend=%s --home=%s --output=json | jq .mnemonic > %s/faucet.mnemonic`,
			sekaiPlugin.sekaidConfig.KeyringBackend, sekaiPlugin.sekaidConfig.SekaidHome, sekaiPlugin.sekaidConfig.MnemonicDir),
		fmt.Sprintf("sekaid add-genesis-account %s 150000000000000ukex,300000000000000test,2000000000000000000000000000samolean,1000000lol --keyring-backend=%s --home=%s",
			constants.ValidatorAccountName, sekaiPlugin.sekaidConfig.KeyringBackend, sekaiPlugin.sekaidConfig.SekaidHome),
		fmt.Sprintf(`sekaid gentx-claim %s --keyring-backend=%s --moniker="%s" --home=%s`,
			constants.ValidatorAccountName, sekaiPlugin.sekaidConfig.KeyringBackend, sekaiPlugin.sekaidConfig.Moniker, sekaiPlugin.sekaidConfig.SekaidHome),
	}

	err = sekaiPlugin.runCommands(ctx, commands)
	if err != nil {
		//  log.Errorf("Initialized container error: %s", err)
		return err
	}
	err = sekaiPlugin.applyNewConfigToml(ctx, sekaiPlugin.getStandardConfigPack())
	if err != nil {
		//  log.Errorf("Can't apply new config, error: %s", err)
		return fmt.Errorf("applying new config error: %w", err)
	}

	err = sekaiPlugin.applyNewAppToml(ctx, sekaiPlugin.getGenesisAppConfig())
	if err != nil {
		//  log.Errorf("Can't apply new app config, error: %s", err)
		return fmt.Errorf("applying new app config error: %w", err)
	}

	//  log.Infof("'sekaid' genesis container '%s' initialized", sekaiPlugin.sekaidConfig.SekaidContainerName)
	return nil
}

func (sekaiPlugin *SekaiPlugin) RunSekaidImageCommand(ctx context.Context, cmd string) error {
	hostFolderPath := filepath.Join(os.Getenv("HOME"), "real-folder-path")
	containerMountPath := "/volumes"

	// Ensure the host folder exists
	const dirPermissions = 0o755
	if err := os.MkdirAll(hostFolderPath, dirPermissions); err != nil {
		panic(fmt.Errorf("error when creating host folder: %w", err))
	}

	command, err := shlex.Split(cmd)
	if err != nil {
		return err
	}

	natRPCPort, err := nat.NewPort("tcp", sekaiPlugin.sekaidConfig.RpcPort)
	if err != nil {
		return err
	}

	natP2PPort, err := nat.NewPort("tcp", sekaiPlugin.sekaidConfig.P2PPort)
	if err != nil {
		return err
	}

	natPrometheusPort, err := nat.NewPort("tcp", sekaiPlugin.sekaidConfig.PrometheusPort)
	if err != nil {
		return err
	}

	containerConfig := &container.Config{
		Image: "ghcr.io/mrlutik/sekin:sekai_v0.3.41",
		Cmd:   command,
		Tty:   true,
		ExposedPorts: nat.PortSet{
			natRPCPort:        struct{}{},
			natP2PPort:        struct{}{},
			natPrometheusPort: struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		AutoRemove: true,
		Binds:      []string{hostFolderPath + ":" + containerMountPath},
		PortBindings: nat.PortMap{
			natRPCPort:        []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: sekaiPlugin.sekaidConfig.RpcPort}},
			natP2PPort:        []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: sekaiPlugin.sekaidConfig.P2PPort}},
			natPrometheusPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: sekaiPlugin.sekaidConfig.PrometheusPort}},
		},
	}

	resp, err := sekaiPlugin.dockerOrchestrator.Cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return err
	}

	if err := sekaiPlugin.dockerOrchestrator.Cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	return nil
}

//	-
//
// -
func (sekaiPlugin *SekaiPlugin) InitJoinSekaid(ctx context.Context) {
	// sekaiPlugin.orhestrator.
}

func (sekaiPlugin *SekaiPlugin) setEmptyValidatorState(ctx context.Context) error {
	emptyState := `
	{
		"height": "0",
		"round": 0,
		"step": 0
	}`
	// TODO
	// mount docker volume to the folder on host machine and do file manipulations inside this folder
	tmpFilePath := "/tmp/priv_validator_state.json"
	err := osutils.CreateFileWithData(tmpFilePath, []byte(emptyState))
	if err != nil {
		return fmt.Errorf("unable to create file <%s>, error: %w", tmpFilePath, err)
	}
	sekaidDataFolder := sekaiPlugin.sekaidConfig.SekaidHome + "/data"
	_, err = sekaiPlugin.dockerOrchestrator.ExecCommandInContainer(ctx, sekaiPlugin.sekaidConfig.SekaidContainerName, fmt.Sprintf(`mkdir %s`, sekaidDataFolder))
	if err != nil {
		return fmt.Errorf("unable to create folder <%s>, error: %w", sekaidDataFolder, err)
	}

	// TODO Rewrite so sekaid plugin will work with volume instead sending/receiving file in data stream to/from container
	err = sekaiPlugin.dockerOrchestrator.SendFileToContainer(ctx, tmpFilePath, sekaidDataFolder, sekaiPlugin.sekaidConfig.SekaidContainerName)
	if err != nil {
		return fmt.Errorf("cannot send %s to container, err: %w", tmpFilePath, err)
	}
	return nil
}

func (sekaiPlugin *SekaiPlugin) runCommands(ctx context.Context, commands []string) error {
	// log := logging.Log
	for _, command := range commands {
		// _, err := sekaiPlugin.dockerOrchestrator.ExecCommandInContainer(ctx, s.config.SekaidContainerName, []string{"bash", "-c", command})
		_, err := sekaiPlugin.dockerOrchestrator.ExecCommandInContainer(ctx, sekaiPlugin.sekaidConfig.SekaidContainerName, command)
		if err != nil {
			//  log.Errorf("Command '%s' execution error: %s", command, err)
			return err
		}
	}

	return nil
}

type // TomlValue represents a configuration value to be updated in the '*.toml' file of the 'sekaid' application.
TomlValue struct {
	Tag   string
	Name  string
	Value string
}

// getStandardConfigPack returns a slice of toml value representing the standard configurations to apply to the 'sekaid' application.
func (sekaiPlugin *SekaiPlugin) getStandardConfigPack() []TomlValue {
	configs := []TomlValue{
		// # CFG [base]
		{Tag: "", Name: "moniker", Value: sekaiPlugin.sekaidConfig.Moniker},
		{Tag: "", Name: "fast_sync", Value: "true"},
		// # CFG [FASTSYNC]
		{Tag: "fastsync", Name: "version", Value: "v1"},
		// # CFG [MEMPOOL]
		{Tag: "mempool", Name: "max_txs_bytes", Value: "131072000"},
		{Tag: "mempool", Name: "max_tx_bytes", Value: "131072"},
		// # CFG [CONSENSUS]
		{Tag: "consensus", Name: "timeout_commit", Value: "10000ms"},
		{Tag: "consensus", Name: "create_empty_blocks_interval", Value: "20s"},
		{Tag: "consensus", Name: "skip_timeout_commit", Value: "false"},
		// # CFG [INSTRUMENTATION]
		{Tag: "instrumentation", Name: "prometheus", Value: "true"},
		// # CFG [P2P]
		{Tag: "p2p", Name: "pex", Value: "true"},
		{Tag: "p2p", Name: "private_peer_ids", Value: ""},
		{Tag: "p2p", Name: "unconditional_peer_ids", Value: ""},
		{Tag: "p2p", Name: "persistent_peers", Value: ""},
		{Tag: "p2p", Name: "seeds", Value: ""},
		{Tag: "p2p", Name: "laddr", Value: fmt.Sprintf("tcp://0.0.0.0:%s", sekaiPlugin.sekaidConfig.P2PPort)},
		{Tag: "p2p", Name: "seed_mode", Value: "false"},
		{Tag: "p2p", Name: "max_num_outbound_peers", Value: "32"},
		{Tag: "p2p", Name: "max_num_inbound_peers", Value: "128"},
		{Tag: "p2p", Name: "send_rate", Value: "65536000"},
		{Tag: "p2p", Name: "recv_rate", Value: "65536000"},
		{Tag: "p2p", Name: "max_packet_msg_payload_size", Value: "131072"},
		{Tag: "p2p", Name: "handshake_timeout", Value: "60s"},
		{Tag: "p2p", Name: "dial_timeout", Value: "30s"},
		{Tag: "p2p", Name: "allow_duplicate_ip", Value: "true"},
		{Tag: "p2p", Name: "addr_book_strict", Value: "true"},
		// # CFG [RPC]
		{Tag: "rpc", Name: "laddr", Value: fmt.Sprintf("tcp://0.0.0.0:%s", sekaiPlugin.sekaidConfig.RpcPort)},
		{Tag: "rpc", Name: "cors_allowed_origins", Value: "[ \"*\" ]"},
	}

	return configs
}

func (sekaiPlugin *SekaiPlugin) getGenesisAppConfig() []TomlValue {
	return []TomlValue{
		{Tag: "state-sync", Name: "snapshot-interval", Value: "1000"},
		{Tag: "state-sync", Name: "snapshot-keep-recent", Value: "2"},
		{Tag: "", Name: "pruning", Value: "nothing"},
		{Tag: "", Name: "pruning-keep-recent", Value: "2"},
		{Tag: "", Name: "pruning-keep-every", Value: "100"},
	}
}

func (sekaiPlugin *SekaiPlugin) applyNewConfigToml(ctx context.Context, configsToml []TomlValue) error {
	// log := logging.Log

	// Adding external p2p address to config
	// This action performed here due to avoiding duplication
	// Genesis and Joiner should both have this configuration
	externalP2PConfig, err := sekaiPlugin.getExternalP2PAddress()
	if err != nil {
		// log.Errorf("Getting external P2P address error: %s", err)
		return err
	}
	configsToml = append(configsToml, externalP2PConfig)

	return sekaiPlugin.applyNewConfig(ctx, configsToml, "config.toml")
}

func (sekaiPlugin *SekaiPlugin) getExternalP2PAddress() (TomlValue, error) {
	// log := logging.Log

	publicIp, err := GetPublicIP() // TODO move method to other package?
	if err != nil {
		// log.Errorf("Getting public IP address error: %s", err)
		return TomlValue{}, err
	}

	return TomlValue{
		Tag:   "p2p",
		Name:  "external_address",
		Value: fmt.Sprintf("tcp://%s:%s", publicIp, sekaiPlugin.sekaidConfig.P2PPort),
	}, nil
}

func (sekaiPlugin *SekaiPlugin) applyNewAppToml(ctx context.Context, configsToml []TomlValue) error {
	return sekaiPlugin.applyNewConfig(ctx, configsToml, "app.toml")
}

// applyNewConfig applies a set of configurations to the 'sekaid' application running in the SekaidManager's container.
func (sekaiPlugin *SekaiPlugin) applyNewConfig(ctx context.Context, configsToml []TomlValue, filename string) error {
	// log := logging.Log

	configDir := fmt.Sprintf("%s/config", sekaiPlugin.sekaidConfig.SekaidHome)

	// log.Infof("Applying new configs to '%s/%s'", configDir, filename)

	// TODO Rewrite so sekaid plugin will work with volume instead sending/receiving file in data stream to/from container
	configFileContent, err := sekaiPlugin.dockerOrchestrator.GetFileFromContainer(ctx, configDir, filename, sekaiPlugin.sekaidConfig.SekaidContainerName)
	if err != nil {
		// log.Errorf("Can't get '%s' file of sekaid application. Error: %s", filename, err)
		return fmt.Errorf("getting '%s' file from sekaid container error: %w", filename, err)
	}

	config := string(configFileContent)
	var newConfig string
	for _, update := range configsToml {
		newConfig, err = SetTomlVar(&update, config)
		if err != nil {
			// log.Errorf("Updating ([%s] %s = %s) error: %s\n", update.Tag, update.Name, update.Value, err)

			// TODO What can we do if updating value is not successful?

			continue
		}

		// log.Infof("Value ([%s] %s = %s) updated successfully\n", update.Tag, update.Name, update.Value)

		config = newConfig
	}

	err = sekaiPlugin.dockerOrchestrator.WriteFileDataToContainer(ctx, []byte(config), filename, configDir, sekaiPlugin.sekaidConfig.SekaidContainerName)
	if err != nil {
		// log.Fatalln(err)
	}

	return nil
}

func GetPublicIP() (string, error) {
	//  log.Infoln("Getting public IP address")

	client := dns.Client{Net: "udp4"}

	getPublicIPFromResponse := func(r *dns.Msg) (string, error) {
		for _, ans := range r.Answer {
			switch ans := ans.(type) {
			case *dns.A:
				// log.Debugf("got `dns.A`: '%v'", ans.A.String())
				return ans.A.String(), nil
			case *dns.TXT:
				// log.Debugf("got `dns.TXT`: '%v'", ans.Txt[0])
				return ans.Txt[0], nil
			}
		}
		return "", ErrExtractingPublicIP
	}

	queryDNS := func(qname string, qtype uint16, server string) (string, error) {
		//  log.Infof("Trying the query '%s' and server '%s'", qname, server)

		message := new(dns.Msg)
		message.SetQuestion(dns.Fqdn(qname), qtype)
		r, _, err := client.Exchange(message, server)
		if err != nil {
			//  log.Errorf("Getting public IP error: %s", err)
			return "", err
		}
		return getPublicIPFromResponse(r)
	}

	ip, err := queryDNS("myip.opendns.com.", dns.TypeA, "resolver1.opendns.com.:53")
	if err == nil {
		return ip, nil
	}

	ip, err = queryDNS("o-o.myaddr.l.google.com.", dns.TypeTXT, "ns1.google.com.:53")
	if err == nil {
		return ip, nil
	}

	return "", ErrGettingPublicIPAddress
}

var (
	ErrExtractingPublicIP     = errors.New("unable to extract public IP address")
	ErrGettingPublicIPAddress = errors.New("can't get the public IP address")
	ErrNetworkDoesNotExist    = errors.New("network does not exist")
)

// SetTomlVar updates a specific configuration value in a TOML file represented by the 'config' string.
// The function takes the 'tag', 'name', and 'value' of the configuration to update and
// returns the updated 'config' string. It ensures that the provided 'value' is correctly
// formatted in quotes if necessary and handles the update of configurations within a specific tag or section.
// The 'tag' parameter allows specifying the configuration section where the 'name' should be updated.
// If the 'tag' is empty ("") or not found, the function updates configurations in the [base] section.
func SetTomlVar(config *TomlValue, configStr string) (string, error) {
	tag := strings.TrimSpace(config.Tag)
	name := strings.TrimSpace(config.Name)
	value := strings.TrimSpace(config.Value)

	//  log.Infof("Trying to update the ([%s] %s = %s)", tag, name, value)

	if tag != "" {
		tag = "[" + tag + "]"
	}

	lines := strings.Split(configStr, "\n")

	tagLine, nameLine, nextTagLine := -1, -1, -1

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if tag == "" && StrStartsWith(trimmedLine, name+" =") {
			// log.Debugf("Found base config '%s' on line: %d", name, i)
			nameLine = i
			break
		}
		if tagLine == -1 && IsSubStr(line, tag) {
			// log.Debugf("Found tag config '%s' on line: %d", tag, i)
			tagLine = i
			continue
		}

		if tagLine != -1 && nameLine == -1 && IsSubStr(line, name+" =") {
			// log.Debugf("Found config '%s' from section '%s' on line: %d", tag, name, i)
			nameLine = i
			continue
		}

		if tagLine != -1 && nameLine != -1 && nextTagLine == -1 && IsSubStr(line, "[") && !IsSubStr(line, tag) {
			// log.Debugf("Found next section after '%s' on line: %d", tag, i)
			nextTagLine = i
			break
		}
	}

	if nameLine == -1 || (nextTagLine != -1 && nameLine > nextTagLine) {
		return "", &ConfigurationVariableNotFoundError{
			VariableName: name,
			Tag:          tag,
		}
	}

	if IsNullOrWhitespace(value) {
		// log.Warnf("Quotes will be added, value '%s' is empty or a seq. of whitespaces\n", value)
		value = fmt.Sprintf("\"%s\"", value)
	} else if StrStartsWith(value, "\"") && StrEndsWith(value, "\"") {
		// log.Warnf("Nothing to do, quotes already present in '%q'\n", value)
	} else if (!StrStartsWith(value, "[")) || (!StrEndsWith(value, "]")) {
		if IsSubStr(value, " ") {
			// log.Warnf("Quotes will be added, value '%s' contains whitespaces\n", value)
			value = fmt.Sprintf("\"%s\"", value)
		} else if (!IsBoolean(value)) && (!IsNumber(value)) {
			// log.Warnf("Quotes will be added, value '%s' is neither a number nor boolean\n", value)
			value = fmt.Sprintf("\"%s\"", value)
		}
	}

	lines[nameLine] = name + " = " + value
	// log.Debugf("New line is: %q", lines[nameLine])

	return strings.Join(lines, "\n"), nil
}

func IsNullOrWhitespace(input string) bool {
	return len(strings.TrimSpace(input)) == 0
}

// IsBoolean checks if the given string represents a valid boolean value ("true" or "false").
func IsBoolean(input string) bool {
	_, err := strconv.ParseBool(input)
	return err == nil
}

// IsNumber checks if the given string represents a valid integer number.
func IsNumber(input string) bool {
	_, err := strconv.ParseInt(input, 0, 64)
	return err == nil
}

// StrStartsWith checks if the given string 's' starts with the specified prefix.
func StrStartsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// StrEndsWith checks if the given string 's' ends with the specified suffix.
func StrEndsWith(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// IsSubStr checks if the specified substring 'substring' exists in the given string 's'.
func IsSubStr(s, substring string) bool {
	return strings.Contains(s, substring)
}

func (e *ConfigurationVariableNotFoundError) Error() string {
	return fmt.Sprintf("the configuration does NOT contain a variable name '%s' occurring after the tag '%s'", e.VariableName, e.Tag)
}

type ConfigurationVariableNotFoundError struct {
	VariableName string
	Tag          string
}
