package p4switch

import (
	"context"
	"github.com/antoninbas/p4runtime-go-client/pkg/client"
	"fmt"
	"io/ioutil"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	p4InfoExt = ".p4info.txt"
	p4BinExt  = ".json"
	p4Path    = "../../cmd/controller/p4/"
)

// Changes the configuration of switch, i.e. program currently executing on it, digests and rules
func (sw *GrpcSwitch) ChangeConfig(ctx context.Context,newConfig *SwitchConfig) error {

	sw.config = newConfig

	if _, err := sw.p4RtC.SaveFwdPipeFromBytes(ctx,sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.InitiateConfig(ctx)
	sw.EnableDigest(ctx)
	time.Sleep(defaultWait)
	if _, err := sw.p4RtC.CommitFwdPipe(ctx); err != nil {
		return err
	}
	return nil
}

func (sw *GrpcSwitch) ChangeConfigSync(ctx context.Context,newConfig *SwitchConfig) error {

	sw.config = newConfig

	if _, err := sw.p4RtC.SetFwdPipeFromBytes(ctx,sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.InitiateConfig(ctx)
	sw.EnableDigest(ctx)

	return nil
}

// Uses API of p4_v1 to add entry into the switch
func (sw *GrpcSwitch) AddTableEntry(ctx context.Context,entry *p4_v1.TableEntry) error {
	if err := sw.p4RtC.InsertTableEntry(ctx,entry); err != nil {
		sw.log.Errorf("Error adding entry: %+v\n%v", entry, err)
		sw.errCh <- err
		return err
	}
	sw.log.Tracef("Added entry: %+v", entry)

	return nil
}

// Uses API of p4_v1 to remove entry from the switch
func (sw *GrpcSwitch) RemoveTableEntry(ctx context.Context,entry *p4_v1.TableEntry) error {
	if err := sw.p4RtC.DeleteTableEntry(ctx,entry); err != nil {
		sw.log.Errorf("Error adding entry: %+v\n%v", entry, err)
		sw.errCh <- err
		return err
	}
	sw.log.Tracef("Added entry: %+v", entry)

	return nil
}

// Returns the actual configuration of the switch, if config is nil (like when the switch is first started) tries to read the configuration from file .yml
func (sw *GrpcSwitch) GetConfig() (*SwitchConfig, error) {
	if sw.config == nil {
		config, err := parseSwConfig(sw.GetName(), sw.initialConfigName)
		if err != nil {
			return nil, err
		}
		sw.config = config
	}
	return sw.config, nil
}

// Parses the configuration of switch sw from file .yml
func parseSwConfig(swName string, configFileName string) (*SwitchConfig, error) {
	configs := make(map[string]SwitchConfig)
	configFile, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(configFile, &configs); err != nil {
		return nil, err
	}
	config := configs[swName]
	if config.Program == "" {
		return nil, fmt.Errorf("Switch config not found in file %s", configFileName)
	}
	return &config, nil
}

// Use this function when switch is started or when the configuration is changed, it gets rules from configuration and add them into the switch
func (sw *GrpcSwitch) InitiateConfig(ctx context.Context) error {
	config, err := sw.GetConfig()
	if err != nil {
		return err
	}
	for _, rule := range config.Rules {
		entry, err := CreateTableEntry(sw, rule)
		if err != nil {
			return err
		}
		sw.log.Infof("Entry: %v", entry) //
		sw.AddTableEntry(ctx,entry)
	}
	return nil
}

func readFileBytes(filePath string) []byte {
	var bytes []byte
	if filePath != "" {
		var err error
		if bytes, err = ioutil.ReadFile(filePath); err != nil {
			log.Fatalf("Error when reading binary from '%s': %v", filePath, err)
		}
	}
	return bytes
}

func (sw *GrpcSwitch) readP4Info() []byte {
	p4Info := p4Path + sw.GetProgramName() + p4InfoExt
	sw.log.Tracef("p4Info %s", p4Info)
	return readFileBytes(p4Info)
}

func (sw *GrpcSwitch) readBin() []byte {
	p4Bin := p4Path + sw.GetProgramName() + p4BinExt
	sw.log.Tracef("p4Bin %s", p4Bin)
	return readFileBytes(p4Bin)
}

// This metod send a GetForwardingPipelineConfig to switch SW and check the response, if no error is thrown means that switch is reachable
func (sw *GrpcSwitch) IsReachable(ctx context.Context) bool {
	_, err := sw.p4RtC.GetFwdPipe(ctx,client.GetFwdPipeCookieOnly)
	return err == nil
}
