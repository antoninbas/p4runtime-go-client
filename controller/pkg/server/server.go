package server

import (
	"controller/pkg/p4switch"
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SwitchServerData struct {
	Name           string
	ProgramName    string
	ProgramActions []p4switch.RuleDescriber
	InstalledRules []p4switch.Rule
}

type RootPageData struct {
	Switches       []SwitchServerData
	ProgramNames   []string
	ErrorMessage   string
	SuccessMessage string
}

type AddRulePageData struct {
	SwitchName string
	Rule       p4switch.RuleDescriber
}

type TopologyJSONData struct {
	Nodes []NodeTopologyData `json:"nodes"`
	Edges []EdgeTopologyData `json:"edges"`
}

const (
	pathP4folder = "../p4/"
	serverPath   = "./pkg/server/"
)

var errorMessage string
var successMessage string

var allSwitches []*p4switch.GrpcSwitch
var programNames []string
var topologyFilePath string

func StartServer(switches []*p4switch.GrpcSwitch, topology string) {
	allSwitches = switches
	topologyFilePath = topology

	http.HandleFunc("/", getRoot)
	http.HandleFunc("/addRule", addRule)
	http.HandleFunc("/removeRule", removeRule)
	http.HandleFunc("/executeProgram", executeProgram)
	http.HandleFunc("/topology", getTopology)
	http.HandleFunc("/getTopologyData", getTopologyData)

	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir(serverPath+"web"))))

	log.Infof("Server listening on localhost:3333")
	err := http.ListenAndServe(":3333", nil)
	if errors.Is(err, http.ErrServerClosed) {
		log.Infof("server closed\n")
	} else if err != nil {
		log.Errorf("Error starting server: %s\n", err)
		os.Exit(1)
	}
}

func getRoot(w http.ResponseWriter, r *http.Request) {

	// Read available programs names

	if programNames == nil {
		files, err := ioutil.ReadDir(pathP4folder)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			fileName := file.Name()
			if !file.IsDir() && strings.HasSuffix(fileName, ".p4") {
				p4ProgramName := fileName[:len(fileName)-len(".p4")]
				programNames = append(programNames, p4ProgramName)
			}
		}
	}

	// Create variable for the rootPage
	var swData []SwitchServerData

	for _, sw := range allSwitches {
		swData = append(swData, SwitchServerData{
			Name:           sw.GetName(),
			ProgramName:    sw.GetProgramName(),
			ProgramActions: getDescribersForSwitch(sw),
			InstalledRules: sw.GetInstalledRules(),
		})
	}

	data := RootPageData{
		Switches:       swData,
		ProgramNames:   programNames,
		SuccessMessage: successMessage,
		ErrorMessage:   errorMessage,
	}

	// compile template
	tmpl := template.Must(template.ParseFiles(serverPath + "index.html"))

	err := tmpl.Execute(w, data)

	if err != nil {
		log.Errorf(err.Error())
	}

	successMessage = ""
	errorMessage = ""
}

func getTopology(w http.ResponseWriter, r *http.Request) {

	tmpl := template.Must(template.ParseFiles(serverPath + "topology.html"))

	err := tmpl.Execute(w, nil)

	if err != nil {
		log.Errorf(err.Error())
	}
}

type NodeTopologyData struct {
	Id    int    `json:"id"`
	Label string `json:"label"`
	Group string `json:"group"`
}
type EdgeTopologyData struct {
	From int `json:"from"`
	To   int `json:"to"`
}

func getTopologyData(w http.ResponseWriter, r *http.Request) {

	var nodes []NodeTopologyData
	nodesMap := make(map[string]int)

	fileData, err := ioutil.ReadFile(topologyFilePath)
	if err != nil {
		errorMessage = "Failed to open topology file!"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	var topo map[string]interface{}
	json.Unmarshal(fileData, &topo)

	i := 1
	for hostName := range topo["hosts"].(map[string]interface{}) {

		nodes = append(nodes, NodeTopologyData{
			Id:    i,
			Label: "Host " + hostName,
			Group: "hosts",
		})
		nodesMap[hostName] = i
		i++
	}

	for _, switchName := range topo["switches"].([]interface{}) {
		sw := getSwitchByName(switchName.(string))
		group := "switches"
		label := "Switch " + switchName.(string)

		if sw == nil || !sw.IsReachable() {
			group = "switchesUnavailable"
			label = label + "\nUNAVAILABLE"
		}

		nodes = append(nodes, NodeTopologyData{
			Id:    i,
			Label: label,
			Group: group,
		})
		nodesMap[switchName.(string)] = i
		i++
	}
	var edges []EdgeTopologyData
	for _, val := range topo["links"].([]interface{}) {
		links := val.([]interface{})

		from := strings.Split(links[0].(string), "-")[0]
		to := strings.Split(links[1].(string), "-")[0]
		edges = append(edges, EdgeTopologyData{
			From: nodesMap[from],
			To:   nodesMap[to],
		})
	}

	data := TopologyJSONData{
		Nodes: nodes,
		Edges: edges,
	}
	jsonResult, _ := json.Marshal(data)

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResult)
}

func addRule(w http.ResponseWriter, r *http.Request) {

	// read parameters from GET request and check validity
	swName := r.URL.Query().Get("switch")
	paramIdAction := r.URL.Query().Get("idAction")
	paramIdTable := r.URL.Query().Get("idTable")

	if swName == "" || paramIdAction == "" || paramIdTable == "" {
		errorMessage = "Failed to add entry: parameters 'switch', 'idAction' and 'idTable' are required"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	idAction, err := strconv.Atoi(paramIdAction)

	idTable, err2 := strconv.Atoi(paramIdTable)

	if err != nil || err2 != nil {
		errorMessage = "Failed to add entry: idAction or idTable are not integer"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	sw := getSwitchByName(swName)
	rule_descr := findActionByIdAndTable(sw, idAction, idTable)

	if sw == nil || rule_descr == nil {
		errorMessage = "Failed to add entry: switch not found or not found rule with specified idAction and idTable"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		// if POST, this is a request for adding a new rule in switch sw

		if err := r.ParseForm(); err != nil {
			log.Error("ParseForm() err:", err)
			return
		}

		// To handle this request:
		// 1) extract informations of rule
		// 2) add new rule to switch sw
		// 3) write a success/failure message on right variable
		// 4) show index page by calling http.Redirect(w, r, "/", http.StatusSeeOther)

		// Extract values of keys from POST and handle errors if one isn't present
		var inputKeys []p4switch.Key
		var inputMask string
		for idx, desc := range rule_descr.Keys {
			if strings.ToUpper(desc.MatchType) == "TERNARY" {
				inputMask = r.FormValue("mask" + strconv.Itoa(idx))
				if inputMask == "" {
					errorMessage = "Failed to add entry: expected mask for key number " + strconv.Itoa(idx) + " but not found"
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}
			}
			actualValue := r.FormValue("key" + strconv.Itoa(idx))
			if actualValue == "" {
				errorMessage = "Failed to add entry: expected a value for key number " + strconv.Itoa(idx) + " but not found"
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			inputKeys = append(inputKeys, p4switch.Key{
				Value: actualValue,
				Mask:  inputMask,
			})
		}

		// Extract values of actionParameters from POST and handle errors if one isn't present
		var inputParam []string
		for idx := range rule_descr.ActionParams {
			actualValue := r.FormValue("par" + strconv.Itoa(idx))
			if actualValue == "" {
				errorMessage = "Failed to add entry: expected a value for parameter number " + strconv.Itoa(idx) + " but not found"
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			inputParam = append(inputParam, actualValue)
		}

		// Create the rule
		rule := p4switch.Rule{
			Table:       rule_descr.TableName,
			Keys:        inputKeys,
			Action:      rule_descr.ActionName,
			ActionParam: inputParam,
		}

		// Add the rule to switch sw and write success/failure message
		res := sw.AddRule(rule)

		if res != nil {
			errorMessage = "Failed to add entry: " + res.Error()
		} else {
			successMessage = "Entry added with success"
		}

		// Show root page
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	if r.Method == "GET" {

		// if GET, this is a request for the page which allows add a new rule

		data := AddRulePageData{
			SwitchName: swName,
			Rule:       *rule_descr,
		}

		tmpl := template.Must(template.ParseFiles(serverPath + "addRule.html"))

		err = tmpl.Execute(w, data)

		if err != nil {
			log.Error(err)
		}
	}
}

func removeRule(w http.ResponseWriter, r *http.Request) {

	// read parameters from GET request and check validity

	swName := r.URL.Query().Get("switch")
	paramNumRule := r.URL.Query().Get("number")

	if swName == "" || paramNumRule == "" {
		errorMessage = "Failed to delete entry: parameters 'number' and 'switch' are required"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	sw := getSwitchByName(swName)
	numRule, err := strconv.Atoi(paramNumRule)

	if sw == nil || err != nil {
		errorMessage = "Failed to delete entry: switch not found or number is not an integer"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Remove the rule from switch SW and write success/failure message
	res := sw.RemoveRule(numRule)

	if res != nil {
		errorMessage = "Failed to delete entry: " + res.Error()
	} else {
		successMessage = "Entry deleted with success"
	}

	// Show root page
	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func executeProgram(w http.ResponseWriter, r *http.Request) {

	// To handle this request:
	// 1) extract parameters from GET request
	// 2) change program in execution on switch
	// 3) write a success/failure message on right variable
	// 4) show index page by calling http.Redirect(w, r, "/", http.StatusSeeOther)

	// read parameters from GET request and check validity
	program := r.URL.Query().Get("program")

	swName := r.URL.Query().Get("switch")

	if program == "" || swName == "" {
		errorMessage = "Cannot change configuration: parameters 'program' and 'switch' are required"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	sw := getSwitchByName(swName)

	if sw == nil || !programValid(program) {
		errorMessage = "Cannot change configuration: switch not found or program not valid"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// change configuration of switch sw (in our case the new one has no digest or rules) and write success/failure message
	err := sw.ChangeConfig(&p4switch.SwitchConfig{
		Program: program,
		Digest:  []string{},
		Rules:   []p4switch.Rule{},
	})

	if err != nil {
		errorMessage = "Cannot change configuration: " + err.Error()
	} else {
		successMessage = "Config updated to " + program
	}

	// show root page
	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func getSwitchByName(name string) *p4switch.GrpcSwitch {
	for _, sw := range allSwitches {
		if sw.GetName() == name {
			return sw
		}
	}
	return nil
}

func findActionByIdAndTable(sw *p4switch.GrpcSwitch, idAction int, idTable int) *p4switch.RuleDescriber {
	if sw == nil || idAction < 0 || idTable < 0 {
		return nil
	}
	for _, action := range getDescribersForSwitch(sw) {
		if action.ActionId == idAction && action.TableId == idTable {
			return &action
		}
	}
	return nil
}

func getDescribersForSwitch(sw *p4switch.GrpcSwitch) []p4switch.RuleDescriber {
	res := *p4switch.ParseP4Info(sw)

	var describers []p4switch.RuleDescriber

	json.Unmarshal([]byte(res), &describers)

	return describers
}

func programValid(name string) bool {
	for _, el := range programNames {
		if el == name {
			return true
		}
	}
	return false
}
