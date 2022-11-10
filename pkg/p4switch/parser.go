package p4switch

import (
	"github.com/antoninbas/p4runtime-go-client/pkg/client"
	"github.com/antoninbas/p4runtime-go-client/pkg/util/conversion"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	log "github.com/sirupsen/logrus"
)

const (
	pattern_ipv4_addr = "ipv4_address"
	pattern_mac_addr  = "mac_address"
	pattern_port      = "port"

	pathJsonInfo = "../../cmd/p4/"
	extJsonInfo  = ".p4.p4info.json"
)

// Define general parser for Keys
type ParserKeys interface {
	parse(key Key, describer FieldDescriber) client.MatchInterface
}

// Specific parser for Keys with matchType: "exact"
type ExactMatchParser struct {
}

func (p *ExactMatchParser) parse(key Key, describer FieldDescriber) client.MatchInterface {

	var field []byte
	var err error

	switch describer.Pattern {
	// Pattern is added in function parseP4Info(), defines if the key satisfies a known pattern and had to be parsed in a specific way
	case pattern_mac_addr:
		{
			field, err = conversion.MacToBinary(key.Value)
			if err != nil {
				log.Errorf("Error parsing match EXACT %s", key)
				return nil
			}
		}
	case pattern_ipv4_addr:
		{
			field, err = conversion.IpToBinary(key.Value)
			if err != nil {
				log.Errorf("Error parsing match EXACT %s", key)
				return nil
			}
		}
	case pattern_port:
		{
			num, err := strconv.ParseInt(key.Value, 10, 64)
			if err != nil {
				log.Errorf("Error parsing match EXACT %s", key)
				return nil
			}
			field, err = conversion.UInt64ToBinaryCompressed(uint64(num))
			if err != nil {
				log.Errorf("Error parsing match EXACT %s", key)
				return nil
			}
		}
	default:
		return nil
	}

	// add to result the key trasformed into []byte
	return client.MatchInterface(&client.ExactMatch{
		Value: field,
	})
}

// Specific parser for Keys with matchType: "lpm"
type LpmMatchParser struct {
}

func (p *LpmMatchParser) parse(key Key, describer FieldDescriber) client.MatchInterface {

	var field []byte
	var lpm int64
	var err error

	switch describer.Pattern {
	case pattern_ipv4_addr:
		{
			values := strings.Split(key.Value, "/")
			if len(values) != 2 {
				log.Errorf("Error parsing match LPM -> %s", key)
				return nil
			}
			field, err = conversion.IpToBinary(values[0])
			if err != nil {
				log.Errorf("Error parsing field %s\n%v", values[0], err)
				return nil
			}
			lpm, err = strconv.ParseInt(values[1], 10, 64)
			if err != nil {
				log.Errorf("Error parsing lpm %d", lpm)
				return nil
			}
		}
	default:
		return nil
		// Match type LPM can only be related to ipv4 addresses
	}

	return client.MatchInterface(&client.LpmMatch{
		Value: field,
		PLen:  int32(lpm),
	})
}

// Specific parser for Keys with matchType: "ternary"
type TernaryMatchParser struct{}

func (p *TernaryMatchParser) parse(key Key, describer FieldDescriber) client.MatchInterface {

	var field []byte
	var mask []byte
	var err error

	switch describer.Pattern {
	case pattern_ipv4_addr:
		{
			field, err = conversion.IpToBinary(key.Value)
			if err != nil {
				log.Errorf("Error parsing field %s\n%v", key.Value, err)
				return nil
			}
			mask, err = hex.DecodeString(key.Mask)
			if err != nil {
				log.Errorf("Error parsing mask %s", key.Mask)
				return nil
			}
		}
	case pattern_mac_addr:
		{
			field, err = conversion.MacToBinary(key.Value)
			if err != nil {
				log.Errorf("Error parsing field %s\n%v", key.Value, err)
				return nil
			}
			mask, err = hex.DecodeString(key.Mask)
			if err != nil {
				log.Errorf("Error parsing mask %s", key.Mask)
				return nil
			}
		}
	default:
		return nil
	}

	return client.MatchInterface(&client.TernaryMatch{
		Value: field,
		Mask:  mask,
	})
}

// A kind of "ParserFactory", returns the parser for the specified matchType (exact | lpm | ternary)

func getParserForKeys(parserType string) ParserKeys {

	switch strings.ToUpper(parserType) {
	case "EXACT":
		return ParserKeys(&ExactMatchParser{})
	case "LPM":
		return ParserKeys(&LpmMatchParser{})
	case "TERNARY":
		return ParserKeys(&TernaryMatchParser{})
	default:
		return nil
	}
}

// Define general parser for ActionParameters
type ParserActionParams interface {
	parse(params []string, describers []FieldDescriber) [][]byte
}

// There's no need to define more than one parser, because ActionParameters are not influenced by matchType
// but to keep everything more general (and for future extensions), had been defined a general structure and a default parser

type DefaultParserActionParams struct{}

func (p *DefaultParserActionParams) parse(params []string, describers []FieldDescriber) [][]byte {

	actionByte := make([][]byte, len(params))
	var field []byte
	var err error

	for idx, par := range params {
		switch describers[idx].Pattern {
		case pattern_mac_addr:
			{
				field, err = conversion.MacToBinary(par)
				if err != nil {
					log.Errorf("Error parsing parameter %s", par)
					return nil
				}
			}
		case pattern_ipv4_addr:
			{
				field, err = conversion.IpToBinary(par)
				if err != nil {
					log.Errorf("Error parsing parameter %s", par)
					return nil
				}
			}
		case pattern_port:
			{
				num, err := strconv.ParseInt(par, 10, 64)
				if err != nil {
					log.Errorf("Error parsing parameter %s", par)
					return nil
				}
				field, err = conversion.UInt64ToBinaryCompressed(uint64(num))
				if err != nil {
					log.Errorf("Error parsing parameter %s", par)
					return nil
				}
			}
		default:
			return nil
		}
		actionByte[idx] = field
	}
	return actionByte
}

// As said before there is only one parser for ActionParameters, so we return that one, regardless of parserType

func getParserForActionParams(parserType string) ParserActionParams {
	return ParserActionParams(&DefaultParserActionParams{})
}

var infoParsed map[string]string

// Return JSON of []RuleDescriber
func ParseP4Info(sw *GrpcSwitch) *string {

	// check if already parsed p4Info of program actually in switch, then return it, if not parse it and save in map
	if infoParsed == nil {
		infoParsed = make(map[string]string)
	}
	info := infoParsed[sw.GetProgramName()]
	if info != "" {
		return &info
	}

	// Define result variable

	var result []RuleDescriber

	actions := sw.p4RtC.GetActions()
	tables := sw.p4RtC.GetTables()

	for _, table := range tables {
		for _, action := range actions {

			// For every table, check all actions to find the ones associated to the table, then add it to result
			if containsAction(table, int(action.Preamble.Id)) {

				// Extract keys
				keys := []FieldDescriber{}
				for _, matchField := range table.MatchFields {
					keys = append(keys, FieldDescriber{
						Name:      matchField.Name,
						Bitwidth:  int(matchField.Bitwidth),
						MatchType: getMatchTypeOf(matchField),
						Pattern:   findIfKnownPattern(matchField.Name, int(matchField.Bitwidth)), // N.B: function findIfKnownPattern
					})
				}

				// Extract keys
				params := []FieldDescriber{}
				for _, param := range action.Params {
					params = append(params, FieldDescriber{
						Name:     param.Name,
						Bitwidth: int(param.Bitwidth),
						Pattern:  findIfKnownPattern(param.Name, int(param.Bitwidth)), // N.B: function findIfKnownPattern
					})
				}

				// Add to result
				result = append(result, RuleDescriber{
					TableName:    table.Preamble.Name,
					TableId:      int(table.Preamble.Id),
					Keys:         keys,
					ActionName:   action.Preamble.Name,
					ActionId:     int(action.Preamble.Id),
					ActionParams: params,
				})
			}
		}
	}
	resInByte, err := json.Marshal(result)

	if err != nil {
		return nil
	}
	res := string(resInByte)
	infoParsed[sw.GetProgramName()] = res

	return &res
}

// Util function, return string representing the matchType
func getMatchTypeOf(field *v1.MatchField) string {
	switch field.GetMatchType() {
	case v1.MatchField_EXACT:
		return "EXACT"
	case v1.MatchField_LPM:
		return "LPM"
	case v1.MatchField_TERNARY:
		return "TERNARY"
	case v1.MatchField_RANGE:
		return "RANGE"
	case v1.MatchField_OPTIONAL:
		return "OPTIONAL"
	case v1.MatchField_UNSPECIFIED:
		return "UNSPECIFIED"
	default:
		return ""
	}
}

// Returns a describer for an already defined rule, basing the research on ActionName and TableName
func getDescriberFor(sw *GrpcSwitch, rule Rule) *RuleDescriber {

	res := *ParseP4Info(sw)

	var describers []RuleDescriber

	json.Unmarshal([]byte(res), &describers)

	for _, descr := range describers {
		if rule.Action == descr.ActionName && rule.Table == descr.TableName {
			return &descr
		}
	}

	return nil
}

// Returns pattern if the field respects a known one, using that the parsers can know how to properly parse the field
func findIfKnownPattern(name string, bitwidth int) string {
	if strings.Contains(strings.ToLower(name), "port") {
		return pattern_port
	}
	if strings.Contains(strings.ToLower(name), "addr") {
		switch bitwidth {
		case 32:
			return pattern_ipv4_addr
		case 48:
			return pattern_mac_addr
		}
	}
	return ""
}

// Util function
func containsAction(table *v1.Table, actionId int) bool {

	for _, ref := range table.ActionRefs {
		if ref.Id == uint32(actionId) {
			return true
		}
	}

	return false
}
