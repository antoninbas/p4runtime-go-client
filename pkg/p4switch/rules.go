package p4switch

import (
	"context"
	"github.com/antoninbas/p4runtime-go-client/pkg/client"
	"fmt"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	//log "github.com/sirupsen/logrus"
)

type Rule struct {
	Table       string
	Keys        []Key
	Action      string
	ActionParam []string `yaml:"action_param"` 
	Describer   *RuleDescriber
}

type Key struct {
	Value string
	Mask  string // optional, used in keys with ternary match
}

type RuleDescriber struct {
	TableName    string
	TableId      int
	Keys         []FieldDescriber
	ActionName   string
	ActionId     int
	ActionParams []FieldDescriber
}

type FieldDescriber struct {
	Name      string
	Bitwidth  int
	MatchType string // optional, used in keys
	Pattern   string // optional, if present the parser will use this to discriminate which function parses this field
}

type SwitchConfig struct {
	Rules   []Rule
	Program string
	Digest  []string
}

// Return rules actually installed into the switch, supposing controller is the only one who can add rules, so returns only rules added by controller
func (sw *GrpcSwitch) GetInstalledRules() []Rule {
	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting config of switch: %v", err)
		return nil
	}
	return config.Rules
}

// Adds a new rule into the switch, both in the array containg the installed rules and in the switch sw
func (sw *GrpcSwitch) AddRule(ctx context.Context,rule Rule) error {
	entry, err := CreateTableEntry(sw, rule)
	if err != nil {
		return err
	}

	if err := sw.AddTableEntry(ctx,entry); err != nil {
		return err
	}

	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting config of switch: %v", err)
		return err
	}
	config.Rules = append(config.Rules, rule)
	return nil
}

// Removes the rule at index "idx" from the switch, both from the array containg the installed rules and from the switch sw
func (sw *GrpcSwitch) RemoveRule(ctx context.Context,idx int) error {
	if idx < 0 || idx >= len(sw.config.Rules) {
		return fmt.Errorf("index not valid")
	}

	// if ask for remove an entry, documentation says that only keys will be considered (ActionParams or other fields will not)
	// so maybe should define a method that not parse ActionParams, is all work not needed
	entry, err := CreateTableEntry(sw, sw.config.Rules[idx])
	if err != nil {
		return err
	}

	if err := sw.RemoveTableEntry(ctx,entry); err != nil {
		return err
	}

	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting config of switch: %v", err)
		return err
	}
	config.Rules = append(config.Rules[:idx], config.Rules[idx+1:]...)
	return nil
}

// Return name of P4 program actually executing into the switch
func (sw *GrpcSwitch) GetProgramName() string {
	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting program name: %v", err)
		return ""
	}
	return config.Program
}

func (sw *GrpcSwitch) GetDigests() []string {
	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting digest list: %v", err)
		return make([]string, 0)
	}
	return config.Digest
}

/// Create a variable of type p4_v1.TableEntry, corrisponding to the rule given by argument.
// Uses funcions of parser.go in order to parse Keys and ActionParameters
func CreateTableEntry(sw *GrpcSwitch, rule Rule) (*p4_v1.TableEntry, error) {

	descr := getDescriberFor(sw, rule)
	if descr == nil {
		return nil, fmt.Errorf("Error getting describer for rule, see log for more info")
	}
	sw.log.Infof("Descr: %v",descr) //
	rule.Describer = descr
	
	sw.log.Infof("Rule: %v", rule.Describer.Keys[0].Name)
	interfaces := parseKeys(rule.Keys, rule.Describer.Keys, rule.Describer.Keys[0].Name)
	sw.log.Infof("Table: %v", rule.Table)
	sw.log.Infof("Table_Alt: %v", rule.Describer.TableName)
	if interfaces == nil {
		return nil, fmt.Errorf("Error parsing keys of rule, see log for more info")
	}
	
	parserActParam := getParserForActionParams("default")
	sw.log.Infof("ParserAct: %v", parserActParam)
	actionParams := parserActParam.parse(rule.ActionParam, rule.Describer.ActionParams)
	if actionParams == nil {
		return nil, fmt.Errorf("Error parsing action parameters of rule, see log for more info")
	}
	sw.log.Infof("ActionParams: %v", actionParams)

	return sw.p4RtC.NewTableEntry(
		rule.Table,
		interfaces,
		sw.p4RtC.NewTableActionDirect(rule.Action, actionParams),
		nil,
	), nil
}

// Util function, gets all the keys of a rule and returns the parsed MatchInterfaces
// This function was modified due to the fact that at the current state (27/10/2022) the client needs a map instead of a slice
func parseKeys(keys []Key, describers []FieldDescriber, tableName string) map[string]client.MatchInterface {
	result := make(map[string]client.MatchInterface, len(keys))
	//result := make([]client.MatchInterface,len(keys))
	for idx, key := range keys {
		parserMatch := getParserForKeys(describers[idx].MatchType)
		result[tableName] = parserMatch.parse(key, describers[idx])
		/*if result[tableName] == nil {
			return nil
		}*/
		fmt.Printf("Key: %v\n", key)
		fmt.Printf("Result: %v\n", result[tableName])
	}
	return result
}

