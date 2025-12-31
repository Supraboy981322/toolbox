package main

/*Just a warning, this code is a bit
 *  dense, but it's not complex, I promise.
 *    It's also slightly repetative, plus most
 *      would probably say it's spaghetti code. I
 *        already broke it up into separate functions
 *          for the different config categories. So, if
 *        		you still don't like it, sorry, I'm not rewriting 
 *          		this. (that's also why it's in a separate file) */

import (
	"os"
	"fmt"
	"io/fs"
	"path/filepath"
	"github.com/charmbracelet/log"
	"github.com/Supraboy981322/gomn"
	elh "github.com/Supraboy981322/ELH"
)

func init() {
	var err error
	log.SetLevel(log.DebugLevel)

	elh.WebDir = "web"

	log.Info("reading config...")
	if config, err = gomn.ParseFile("config.gomn"); err != nil {
		log.Fatalf("failed to read config:  %v", err)
	} else { log.Debug("read config")	}

	if ok, errStr := parseTopLevelConf(); !ok {
		log.Fatal(errStr)
	} else { log.Debug("parseTopLevelConf() returned ok") }

	if ok, errStr := parseDashConf(); !ok {
		log.Fatal(errStr)
	} else { log.Debug("parseDashConf() returned ok") }

	//not used yet, but maps custom endpoints
	if ok, errStr := parseEndPtConf(); !ok {
		log.Fatal(errStr)
	} else { log.Debug("parseEndPtConf() returned ok") }
 
	if ok, errStr := parseAdvancedConf(); !ok {
		log.Fatal(errStr)
	} else { log.Debug("parseAdvancedConf() returned ok") }

	if tmpWebDir, err = dumpEmbededFStoDisk(); err != nil {
		log.Fatalf("failed to write embeded filesystem to disk:  %v", err)
	} else { log.Debug("dumped embeded filesystem to disk") }

	log.Info("startup done.")
}

func parseAdvancedConf() (bool, string) {

	if advancedConf, ok := config["advanced"].(gomn.Map); ok {
		log.Debug("using advanced configs options")
		if elhConf, ok := advancedConf["elh"].(gomn.Map); ok {
			log.Debug("using elh configs options")
			if supELHerrs, ok := elhConf["suppress ELH errors"].(bool); ok {
				if supELHerrs {
					log.Warn("suppressing ELH errrors")
					elh.Log.Runner.LogStderr = true
				} else { log.Debug("not suppressing elh errors") }
			}
		} else { log.Debug("not using elh config options") }
	} else { log.Debug("not using advanced config options") }

	return true, ""
}

func parseEndPtConf() (bool, string) {
	ptMapTmp := make(map[string]map[string]string)
	if endPtsRaw, ok := config["endpoints"].(gomn.Map); ok {
		log.Debug("found custom endpoints")

		for ptRaw, mpRaw := range endPtsRaw {
			ptMap := make(map[string]string)

			var mp gomn.Map
			if mp, ok = mpRaw.(gomn.Map); !ok {
				return false, "failed to assert endpoint map to a map"
			} else { log.Debug("asserted endpoint map") } 

			for keyRaw, valRaw := range mp {
				if key, ok := keyRaw.(string); ok {
					if valS, ok := valRaw.(string); ok {
						ptMap[key] = valS
					}	else {
						if valR, ok := valRaw.(string); ok {
							ptMap[key] = string(valR)
						} else { return false, fmt.Sprintf("bad endpoint map value: %v", valRaw) }
					}
				} else { return false, fmt.Sprintf("invalid endpoint map key:  %v", keyRaw) }
			}

			if pt, ok := ptRaw.(string); ok {
				ptMapTmp[pt] = ptMap
			} else { return false, fmt.Sprintf("endpoint not a string:  %v", ptRaw) }

		}; endPtMap = ptMapTmp
	} else { log.Debug("no custom endpoints defined") }
	
	return true, ""
}

func parseDashConf() (bool, string) {
	//check if dashboard is enabled
	//  (I know, this looks highly compressed... because it is)
	if dashBoard, ok := config["dashboard"].(gomn.Map); ok {
		//checked at end of func
		serverName, _ = dashBoard["name"].(string)

		if useWebUI, ok = dashBoard["enable"].(bool); !ok {
			return false, "value of \"enable\" in the dashboard config is not a bool"
		} else if useWebUI {
			log.Debug("dashboard is enabled")
		} else { log.Warn("web ui is disabled") }
	} else { return false, "dashboard config is not a map" } 

	if serverName == "" {
		serverName = "toolbox"
		log.Warn("server is not named, defaulting to "+
					"\""+serverName+"\" (a very creative and unique name)")
	}

	return true, ""
}

func parseTopLevelConf() (bool, string) {
	var ok bool

	var deLvl string
	if deLvl, ok = config["log level"].(string); ok {
		switch deLvl {
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info": 
			log.SetLevel(log.InfoLevel)
		case "warn": 
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		case "fatal":
			log.SetLevel(log.FatalLevel)
		default:
			return false, "invalid log level"
		}

		log.Infof("log level set to:  %s", deLvl)
	} else { return false, "failed to get log level" }

	//set the port from config
	if port, ok = config["port"].(int); !ok {
		return false, "server port is not an integer"
	} else { log.Debug("success reading server port") }

	return true, ""
}

func dumpEmbededFStoDisk() (string, error) {
	var err error
	tmp := "web"
	err = fs.WalkDir(webUIdir, "web", func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }

		dest := filepath.Join(".", path)

		if d.IsDir() { return os.MkdirAll(dest, 0755) }

		dat, err := webUIdir.ReadFile(path)
		if err != nil { return err }
		return os.WriteFile(dest, dat, 0644)
	})

	if err != nil {
		os.RemoveAll(tmp)
		return "", err
	}

	return tmp, nil
}
