package main

/* an automatic minecraft modpack manager
works on a standard Flame manifest.json
usage: modpkg [workdir]
if workdir not specified, works in cwd */

import (
	"encoding/json"
	"archive/zip"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"io"
	"sync"
	//    "sort"
	"path/filepath"
)

type Manifest struct {
	Minecraft       map[string]interface{}   `json:"minecraft"`
	Files           []map[string]interface{} `json:"files"`
	Author          string                   `json:"author"`
	ManifestType    string                   `json:"manifestType"`
	ManifestVersion int                      `json:"manifestVersion"`
	Name            string                   `json:"name"`
	Version         string                   `json:"version"`
	Overrides       string                   `json:"overrides"`
}

type Mod struct {
	ProjectID int                      `json:"id"`
	GVLF      []map[string]interface{} `json:"gameVersionLatestFiles"`
	Name      string                   `json:"name"`
}

func HandleMod(index int, files *[]map[string]interface{}, mcver string, wg *sync.WaitGroup, client *http.Client, isForge bool) {
	projectid := int((*files)[index]["projectID"].(float64)) /* reasons */
	res, err := client.Get(fmt.Sprint("https://addons-ecs.forgesvc.net/api/v2/addon/", projectid))
	if err != nil {
		log.Fatal(projectid, " : ", err)
	}
	defer res.Body.Close()
	resbody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(projectid, " : ", err)
	}
	var mod Mod
	_ = json.Unmarshal(resbody, &mod)
	fmt.Printf("%d : handling %s\n", projectid, mod.Name)
	var fileid int
FindVer:
	for _, v := range mod.GVLF {
		if v["gameVersion"].(string) == mcver {
			fileidtemp := int(v["projectFileId"].(float64))
			if isForge {
				fmt.Printf("%d : found file id %d (%s)\n", projectid, fileidtemp, mod.Name)
				fileid = fileidtemp
				break
			} else {
				res2, err := client.Get(fmt.Sprintf("https://addons-ecs.forgesvc.net/api/v2/addon/%d/file/%d", projectid, fileidtemp))
				if err != nil {
					log.Fatal(projectid, " : ", err)
				}
				resbody2, err := io.ReadAll(res2.Body)
				if err != nil {
					log.Fatal(projectid, " : ", err)
				}
				var file interface{}
				err = json.Unmarshal(resbody2, &file)
				if err != nil {
					fail := string(resbody2)
					fmt.Println("fail on ", fail)
					log.Fatal(projectid, " : ", err)
				}
				defer res2.Body.Close()
				datastuff := file.(map[string]interface{})
				gameversions := datastuff["gameVersion"].([]interface{})
				for _, v := range gameversions {
					if v.(string) == "Forge" {
						fmt.Printf("%d : found file id %d (%s)\n", projectid, fileidtemp, mod.Name)
						fileid = fileidtemp
						break FindVer
					}
				}
			}
		}
	}
	if fileid == 0 {
		log.Fatalf("%d : no appropriate file found (searched for %s)", projectid, mcver)
	}
	(*files)[index]["fileID"] = fileid;
	wg.Done()
}

func main() {
	var workdir string
	switch len(os.Args) {
	case 1:
		var err error
		workdir, err = os.Getwd()
		if err != nil {
			log.Fatalf("lolwut: %s", err)
		}
	case 2:
		workdir = os.Args[1]

	default:
		log.Fatal("weird number of args")
	}
	if err := os.Chdir(workdir); err != nil {
		log.Fatal(err)
	}
	manifest, err := os.ReadFile("manifest.json")
	if err != nil {
		log.Fatal(err)
	}
	var i Manifest
	var wg sync.WaitGroup
	client := &http.Client{}
	_ = json.Unmarshal(manifest, &i)
	mcver := i.Minecraft["version"].(string)
	for index, _ := range i.Files {
		wg.Add(1)
		var modver string
		if i.Files[index]["modpkgver"] != nil {
			modver = i.Files[index]["modpkgver"].(string)
		} else {
			modver = mcver
		}
		isForge := false
		if i.Files[index]["modpkgIsForge"] == true {
			isForge = true
		}
		go HandleMod(index, &i.Files, modver, &wg, client, isForge)
	}
	wg.Wait()
	out, err := json.Marshal(i)
	if err != nil {
		log.Fatal(err)
	}
	zipName := strings.ReplaceAll(i.Name, " ", "+")
	modpack, err := os.Create(fmt.Sprintf("%s-%s.zip", zipName, i.Version))
	if err != nil {
		log.Fatal(err)
	}
	defer modpack.Close()
	
	writer := zip.NewWriter(modpack)
	defer writer.Close()
	
	manifestjson, err := writer.Create("manifest.json")
	if err != nil {
		log.Fatal(err)
	}
	_, err = manifestjson.Write(out)
	if err != nil {
		log.Fatal(err)
	}
	
	overridewalker := func(path string, info os.FileInfo, err error) error {
		fmt.Printf("Packing up %s\n", path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		f, err := writer.Create(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}
		return nil
	}
	err = filepath.Walk(i.Overrides, overridewalker)
	if err != nil {
		log.Fatal(err)
	}
}
