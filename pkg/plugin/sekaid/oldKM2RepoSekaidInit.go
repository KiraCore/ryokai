package sekaid

import (
	"bufio"
	"context"
	"fmt"
	"os"

	osutils "github.com/KiraCore/ryokai/pkg/ryokaicommon/utils/os"
	vlg "github.com/PeepoFrog/validator-key-gen/MnemonicsGenerator"
	"github.com/joho/godotenv"
	kiraMnemonicGen "github.com/kiracore/tools/bip39gen/cmd"
	"github.com/kiracore/tools/bip39gen/pkg/bip39"
)

func (sekaiPlugin *SekaiPlugin) ReadMnemonicsFromFile(pathToFile string) (masterMnemonic string, err error) {
	// log := logging.Log
	// log.Infof("Checking if path exist: %s", pathToFile)
	check := osutils.PathExists(pathToFile)

	if check {
		// log.Infof("Path exist, trying to read mnemonic from mnemonics.env file")
		if err := godotenv.Load(pathToFile); err != nil {
			err = fmt.Errorf("error loading .env file: %w", err)
			return "", err
		}
		// Retrieve the MASTER_MNEMONIC value
		const masterMnemonicEnv = "MASTER_MNEMONIC"
		masterMnemonic = os.Getenv(masterMnemonicEnv)
		if masterMnemonic == "" {
			err = &EnvVariableNotFoundError{VariableName: masterMnemonicEnv}
			return masterMnemonic, err
		} else {
			// log.Debugf("MASTER_MNEMONIC: %s", masterMnemonic)
		}
	}

	return masterMnemonic, nil
}

func (e *EnvVariableNotFoundError) Error() string {
	return fmt.Sprintf("env variable '%s' not found", e.VariableName)
}

type EnvVariableNotFoundError struct {
	VariableName string
}

func (sekaiPlugin *SekaiPlugin) GenerateMnemonicsFromMaster(masterMnemonic string) (*vlg.MasterMnemonicSet, error) {
	// log := logging.Log
	// log.Debugf("GenerateMnemonicFromMaster: masterMnemonic:\n%s", masterMnemonic)
	defaultPrefix := "kira"
	defaultPath := "44'/118'/0'/0/0"

	mnemonicSet, err := vlg.MasterKeysGen([]byte(masterMnemonic), defaultPrefix, defaultPath, sekaiPlugin.sekaidConfig.SecretsFolder)
	if err != nil {
		return &vlg.MasterMnemonicSet{}, err
	}
	// str := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", mnemonicSet.SignerAddrMnemonic, mnemonicSet.ValidatorNodeMnemonic, mnemonicSet.ValidatorNodeId, mnemonicSet.ValidatorAddrMnemonic, mnemonicSet.ValidatorValMnemonic)
	// log.Infof("Master mnemonic:\n%s", str)
	return &mnemonicSet, nil
}

func (sekaiPlugin *SekaiPlugin) MnemonicReader() (masterMnemonic string) {
	// log := logging.Log
	// log.Infoln("ENTER YOUR MASTER MNEMONIC:")

	reader := bufio.NewReader(os.Stdin)
	//nolint:forbidigo // reading user input
	fmt.Println("Enter mnemonic: ")

	text, err := reader.ReadString('\n')
	if err != nil {
		// log.Errorf("An error occurred: %s", err)
		return
	}
	mnemonicBytes := []byte(text)
	mnemonicBytes = mnemonicBytes[0 : len(mnemonicBytes)-1]
	masterMnemonic = string(mnemonicBytes)
	return masterMnemonic
}

// GenerateMnemonic generates random bip 24 word mnemonic
func (sekaiPlugin *SekaiPlugin) GenerateMnemonic() (masterMnemonic bip39.Mnemonic, err error) {
	masterMnemonic = kiraMnemonicGen.NewMnemonic()
	masterMnemonic.SetRandomEntropy(24)
	masterMnemonic.Generate()

	return masterMnemonic, nil
}

func (sekaiPlugin *SekaiPlugin) SetSekaidKeys(ctx context.Context) error {
	// TODO path set as variables or constants
	// log := logging.Log
	sekaidConfigFolder := sekaiPlugin.sekaidConfig.SekaidHome + "/config"
	_, err := sekaiPlugin.dockerOrchestrator.ExecCommandInContainer(ctx, sekaiPlugin.sekaidConfig.SekaidContainerName, fmt.Sprintf(`mkdir %s`, sekaiPlugin.sekaidConfig.SekaidHome))
	if err != nil {
		return fmt.Errorf("unable to create <%s> folder, err: %w", sekaiPlugin.sekaidConfig.SekaidHome, err)
	}
	_, err = sekaiPlugin.dockerOrchestrator.ExecCommandInContainer(ctx, sekaiPlugin.sekaidConfig.SekaidContainerName, fmt.Sprintf(`mkdir %s`, sekaidConfigFolder))
	if err != nil {
		return fmt.Errorf("unable to create <%s> folder, err: %w", sekaidConfigFolder, err)
	}

	// TODO: REWORK, REMOVE SendFileToContainer and work with volume
	err = sekaiPlugin.dockerOrchestrator.SendFileToContainer(ctx, sekaiPlugin.sekaidConfig.SecretsFolder+"/priv_validator_key.json", sekaidConfigFolder, sekaiPlugin.sekaidConfig.SekaidContainerName)
	if err != nil {
		// log.Errorf("cannot send priv_validator_key.json to container\n")
		return err
	}

	err = osutils.CopyFile(sekaiPlugin.sekaidConfig.SecretsFolder+"/validator_node_key.json", sekaiPlugin.sekaidConfig.SecretsFolder+"/node_key.json")
	if err != nil {
		// log.Errorf("copying file error: %s", err)
		return err
	}

	err = sekaiPlugin.dockerOrchestrator.SendFileToContainer(ctx, sekaiPlugin.sekaidConfig.SecretsFolder+"/node_key.json", sekaidConfigFolder, sekaiPlugin.sekaidConfig.SekaidContainerName)
	if err != nil {
		// log.Errorf("cannot send node_key.json to container")
		return err
	}
	return nil
}

// sets empty state of validator into $sekaidHome/data/priv_validator_state.json
func (sekaiPlugin *SekaiPlugin) SetEmptyValidatorState(ctx context.Context) error {
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
	// _, err = sekaiPlugin.dockerOrchestrator.ExecCommandInContainer(ctx, sekaiPlugin.sekaidConfig.SekaidContainerName, []string{"bash", "-c", fmt.Sprintf(`mkdir %s`, sekaidDataFolder)})
	_, err = sekaiPlugin.dockerOrchestrator.ExecCommandInContainer(ctx, sekaiPlugin.sekaidConfig.SekaidContainerName, fmt.Sprintf(`mkdir %s`, sekaidDataFolder))
	if err != nil {
		return fmt.Errorf("unable to create folder <%s>, error: %w", sekaidDataFolder, err)
	}
	err = sekaiPlugin.dockerOrchestrator.SendFileToContainer(ctx, tmpFilePath, sekaidDataFolder, sekaiPlugin.sekaidConfig.SekaidContainerName)
	if err != nil {
		return fmt.Errorf("cannot send %s to container, err: %w", tmpFilePath, err)
	}
	return nil
}
