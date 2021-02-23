package main

import (
	"bufio"
	"fmt"
	helpers "go_patching/functions"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// Globals
var ROOTPATH string = helpers.GetScriptRootPath()
var PATCHED_FILES_DIR string = filepath.Join(ROOTPATH, "files")
var MANIFEST_CONFIG string = filepath.Join(ROOTPATH, "manifest.cfg")
var MANIFEST string = filepath.Join(ROOTPATH, "manifest")
var BACKUP string = filepath.Join(ROOTPATH, "backup")
var num_files int = 0

// Constants
const GOOS string = runtime.GOOS

func menu() string {
	fmt.Printf("Patching Utility Main Menu (%s)\n", GOOS)
	fmt.Println("\t1. Build")
	fmt.Println("\t2. Patch")
	fmt.Println("\t3. Restore")
	fmt.Println("\t4. Exit")
	fmt.Printf("Select an option: ")
	var choice string
	fmt.Scanln(&choice)
	return choice
}

func fileExists(f string) bool {
	if _, err := os.Stat(f); err == nil {
		return true // file exists
	}
	return false // file does not exist
}

// Checks if this line is a file or directory, including children, and
// calls writeToManifest()
func analyzeLine(line string) {
	fileInfo, err := os.Stat(line)
	if err != nil {
		fmt.Println(err)
		return
	}
	switch mode := fileInfo.Mode(); {

	case mode.IsDir():
		md5 := "-"
		writeToManifest(md5, line, fileInfo)
		// process all children
		scanDir(line)

	case mode.IsRegular():
		md5, _ := helpers.HashFileMD5(line)
		writeToManifest(md5, line, fileInfo)
		// absolute path to patched file
		dest := filepath.Join(PATCHED_FILES_DIR, line)
		// Get parent dir perms
		parent := filepath.Dir(line)
		parentInfo, _ := os.Stat(parent)
		// parent directory located in patch
		patch_parents := filepath.Join(PATCHED_FILES_DIR, parent)
		// create directory structure
		os.MkdirAll(patch_parents, parentInfo.Mode())
		// copy file to the patch
		helpers.CopyFile(line, dest)
	}
}

// Processes all contents of a directory
func scanDir(line string) {
	fmt.Printf("scanning %s\n", line)
	files, err := ioutil.ReadDir(line)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		// process all children in this directory
		analyzeLine(filepath.Join(line, f.Name()))
	}
}

// Writes information to manifest
func writeToManifest(md5 string, src string, fileInfo fs.FileInfo) {
	lineToPrint := ""

	// Information differs based on OS
	if GOOS == "linux" {
		file_sys := fileInfo.Sys()
		uid := fmt.Sprint(file_sys.(*syscall.Stat_t).Uid)
		gid := fmt.Sprint(file_sys.(*syscall.Stat_t).Gid)
		perms := fmt.Sprintf("%04o", fileInfo.Mode().Perm())
		lineToPrint = fmt.Sprintf("%s,%s,%s,%s,%s\n", md5, src, uid, gid, perms)
	} else if GOOS == "windows" {
		//patchSrc = StripDriveLetterFromPath(src)
		//lineToPrint = fmt.Sprintf("%s,%s,%s\n")
	}

	// Create and open manifest file
	f, err := os.OpenFile(MANIFEST, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	// Write the line
	if _, err := f.WriteString(lineToPrint); err != nil {
		log.Println(err)
	} else {
		fmt.Printf("added %s to manifest.\n", src)
	}
}

// Opens and reads each line of manifest.cfg
func readManifestConfig(cfg *string) {
	file, err := os.Open(*cfg)

	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			analyzeLine(line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func removeAndCreateDirectory(d string) {
	if fileExists(d) {
		// remove
		err := os.RemoveAll(d)
		if err != nil {
			log.Fatal(err)
		}
	}
	// create
	os.Mkdir(d, os.ModePerm)
}

func build() {
	if fileExists(MANIFEST_CONFIG) {
		// Cleanup any leftover files and directories from previous builds
		removeAndCreateDirectory(BACKUP)
		os.RemoveAll(PATCHED_FILES_DIR)
		os.Remove(MANIFEST)
		os.Create(MANIFEST)
		readManifestConfig(&MANIFEST_CONFIG)
	} else {
		fmt.Println("Manifest config file does not exist.")
	}
	fmt.Println("")
}

func patch() {
	if fileExists(MANIFEST) {
		readManifest()
	} else {
		fmt.Println("manifest not found.")
	}
	fmt.Println("")
}

func parseLine(line string) {
	data := strings.Split(line, ",")

	m := make(map[string]string)

	if GOOS == "linux" {
		m["md5"] = data[0]
		m["dest"] = data[1]
		m["uid"] = data[2]
		m["gid"] = data[3]
		m["perms"] = data[4]

		compareOrReplace(m)
	} else if GOOS == "windows" {

	}
}

func readManifest() {
	file, err := os.Open(MANIFEST)

	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			parseLine(line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func compareOrReplace(m map[string]string) {
	src := filepath.Join(PATCHED_FILES_DIR, m["dest"])
	fileinfo, err := os.Stat(src)
	if err != nil {
		fmt.Println(err)
		return
	}

	if !fileExists(m["dest"]) {
		// replace the file
		fmt.Printf("missing file %s, replacing...\n", m["dest"])
		patchFile(src, m, fileinfo)
	} else {
		// compare hashes
		destHash, _ := helpers.HashFileMD5(m["dest"])
		if m["md5"] != destHash && !fileinfo.IsDir() {
			// hashes do not match, backup and replace the file
			fmt.Printf("patching %s\n", m["dest"])
			patchFile(src, m, fileinfo)
		}
	}
}

func patchFile(src string, m map[string]string, fileinfo fs.FileInfo) {
	switch mode := fileinfo.Mode(); {
	case mode.IsDir():
		helpers.CopyDir(src, m["dest"])
	case mode.IsRegular():
		helpers.CopyFile(src, m["dest"])
	}

	if GOOS == "linux" {
		// set owner and permissions
		uid, _ := strconv.Atoi(m["uid"])
		gid, _ := strconv.Atoi(m["gid"])
		perms, _ := strconv.ParseUint(m["perms"], 8, 32)
		err := os.Chown(m["dest"], uid, gid)
		if err != nil {
			log.Fatal(err)
		}
		err = os.Chmod(m["dest"], os.FileMode(perms))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	// Display main menu
	for true {
		switch action := menu(); action {
		case "1":
			// Build a patch
			fmt.Printf("\nbuilding patch...\n")
			build()
		case "2":
			fmt.Printf("\npatching files...\n")
			patch()
		case "3":
			fmt.Println("Restore")
		case "4":
			os.Exit(0)
		default:
			fmt.Printf("\nInvalid option, try again!\n\n")
		}
	}
}
