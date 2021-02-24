/*
Title: Patching Utility
Author: Nick Marriotti
Date: 2/24/2021
*/

package main

import (
	"fmt"
	helpers "go_patching/pkg/functions"
	"go_patching/pkg/osspecific"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// File locations
var ROOTPATH string = helpers.GetScriptRootPath()
var PATCHED_FILES_DIR string = filepath.Join(ROOTPATH, "files")
var MANIFEST_CONFIG string = filepath.Join(ROOTPATH, "manifest.cfg")
var MANIFEST string = filepath.Join(ROOTPATH, "manifest")
var BACKUP string = filepath.Join(ROOTPATH, "backup")

// Will track number of files updated or changed
var num_files int = 0

// Operating system being used logic adjusts based on this
const GOOS string = runtime.GOOS

// Main menu
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

// Copies file and parent directory structure to an alternate location.
func BackupFile(src string, destfolder string) bool {
	fileInfo, err := os.Stat(src)
	if err != nil {
		fmt.Println(err)
		return false
	}

	// Will be populated based on OS
	dest := ""
	parent := ""
	backup_parents := ""

	if GOOS == "linux" {
		dest = filepath.Join(destfolder, src)
		parent = filepath.Dir(src)
		backup_parents = filepath.Join(destfolder, parent)

	} else if GOOS == "windows" {

		/* The following comments give examples of what each function is doing.
		   Windows is a bit different because of the drive letters */

		// C:\Users\admin\test.txt -> Users\admin\test.txt
		updated_winpath := helpers.RemoveWindowsDriveLetterFromPath(src)
		// C:\Users\admin\patch\files\Users\admin\test.txt
		dest = filepath.Join(destfolder, updated_winpath)
		// C:\Users\admin\
		parent = filepath.Dir(src)
		// C:\Users\admin\patch\files\Users\admin
		backup_parents = filepath.Join(destfolder, filepath.Dir(updated_winpath))
	}

	switch mode := fileInfo.Mode(); {

	case mode.IsDir():
		fmt.Printf("creating directory backup of %s\n", src)
		helpers.CopyDir(src, dest)

	case mode.IsRegular():
		// Get information about this file
		parentInfo, _ := os.Stat(parent)
		fmt.Printf("creating file backup of %s\n", src)
		// create directory structure
		os.MkdirAll(backup_parents, parentInfo.Mode())
		// copy file to the patch
		helpers.CopyFile(src, dest)
	}

	return true
}

// Checks if this line is a file or directory, including children, and
// calls writeToManifest()
func AnalyzeLine(line string) {
	fileInfo, err := os.Stat(line)
	if err != nil {
		fmt.Println(err)
		return
	}

	switch mode := fileInfo.Mode(); {

	case mode.IsDir():
		md5 := "-"
		WriteToManifest(md5, line, fileInfo)
		ScanDir(line)

	case mode.IsRegular():
		md5, _ := helpers.HashFileMD5(line)
		WriteToManifest(md5, line, fileInfo)
		BackupFile(line, PATCHED_FILES_DIR)
	}
}

// Finds and checks all contents of a directory
func ScanDir(line string) {
	fmt.Printf("scanning %s\n", line)
	files, err := ioutil.ReadDir(line)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		// process all children in this directory
		AnalyzeLine(filepath.Join(line, f.Name()))
	}
}

// Writes information to manifest
func WriteToManifest(md5 string, src string, fileInfo fs.FileInfo) {
	lineToPrint := ""

	// Information differs based on OS
	if GOOS == "linux" {
		f_info := osspecific.GetLinuxFileInfo(fileInfo)
		lineToPrint = fmt.Sprintf("%s,%s,%s,%s,%s\n", md5, src, f_info[0], f_info[1], f_info[2])
	} else if GOOS == "windows" {
		lineToPrint = fmt.Sprintf("%s,%s\n", md5, src)
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
		// increment number of files included in the patch
		num_files += 1
	}
}

// Builds a patch based on what is specified in manifest.cfg
func Build() {
	// reset
	num_files = 0

	if helpers.FileExists(MANIFEST_CONFIG) {
		fmt.Printf("building...\n")
		// Cleanup any leftover files and directories from previous builds
		os.RemoveAll(BACKUP)
		os.RemoveAll(PATCHED_FILES_DIR)
		os.Remove(MANIFEST)
		os.Create(MANIFEST)

		// Get all lines in manifest.cfg
		lines := helpers.ReadFile(MANIFEST_CONFIG)
		// process each line
		for _, line := range lines {
			AnalyzeLine(line)
		}

	} else {
		fmt.Println("manifest config file not found.")
	}

	if num_files > 0 {
		fmt.Printf("build complete. added %d files\n", num_files)
	} else {
		fmt.Println("complete. no files found.")
	}
	fmt.Println("")
}

// Restores a patched file to its unpatched state
func Restore() {
	// reset restore counter
	num_files = 0

	// read manifest lines
	if helpers.FileExists(MANIFEST) {
		fmt.Printf("restoring...\n")
		lines := helpers.ReadFile(MANIFEST)
		for _, line := range lines {
			_map := StringToMap(line)
			// add src key to point to file stored in backup directory
			dest := _map["dest"]

			src := ""
			if GOOS == "linux" {
				src = filepath.Join(BACKUP, dest)
			} else if GOOS == "windows" {
				// builds absolute path to where this file resides in the backup directory
				// and removes the drive letter to avoid multiple drive references from
				// appearing in the src path
				src = filepath.Join(BACKUP, helpers.RemoveWindowsDriveLetterFromPath(dest))
			}

			if helpers.FileExists(src) {
				if HashMismatch(src, dest) {
					// replace the file
					fmt.Printf("restoring %s\n", dest)
					helpers.CopyFile(src, dest)
					// track number of files restored
					num_files += 1
				}
			}
		}

		if num_files > 0 {
			fmt.Printf("complete. restored %d files\n\n", num_files)
		} else {
			fmt.Printf("complete. all files are already in their original state.\n\n")
		}

	} else {
		fmt.Println("manifest not found")
	}
}

// Compares two MD5 hash values and returns true if no match is found
// otherwise returns false
func HashMismatch(file1 string, file2 string) bool {
	md5_1, err := helpers.HashFileMD5(file1)
	if err != nil {
		return false
	}
	md5_2, err := helpers.HashFileMD5(file2)
	if err != nil {
		return false
	}
	if md5_1 != md5_2 {
		return true
	}
	return false
}

// Updates files and directories on the filesystem with what is stored in the patch
func Patch() {
	// reset backups every time the system is patched
	os.RemoveAll(BACKUP)

	// reset patch counter
	num_files = 0

	if helpers.FileExists(MANIFEST) {
		fmt.Printf("patching...\n")
		lines := helpers.ReadFile(MANIFEST)
		for _, line := range lines {
			_map := StringToMap(line)
			CompareOrReplace(_map)
		}

	} else {
		fmt.Println("manifest not found.")
	}

	if num_files > 0 {
		fmt.Printf("complete. patched %d files\n\n", num_files)
	} else {
		fmt.Printf("complete. all files are intact and no action was taken.\n\n")
	}
}

// Parses comma-delimited string into a map
func StringToMap(line string) map[string]string {
	data := strings.Split(line, ",")

	m := make(map[string]string)

	if GOOS == "linux" {
		m["md5"] = data[0]
		m["dest"] = data[1]
		m["uid"] = data[2]
		m["gid"] = data[3]
		m["perms"] = data[4]

	} else if GOOS == "windows" {
		m["md5"] = data[0]
		m["dest"] = data[1]
	}
	return m
}

// Patches files that are missing or have incorrect hash values
func CompareOrReplace(m map[string]string) {
	src := ""

	if GOOS == "linux" {
		src = filepath.Join(PATCHED_FILES_DIR, m["dest"])
	} else if GOOS == "windows" {
		src = filepath.Join(PATCHED_FILES_DIR, helpers.RemoveWindowsDriveLetterFromPath(m["dest"]))
	}

	fileinfo, err := os.Stat(src)
	if err != nil {
		fmt.Println(err)
		return
	}

	if !helpers.FileExists(m["dest"]) {
		// replace the file
		fmt.Printf("unable to backup %s. this file or directory is not present on the system.\n", m["dest"])
		PatchFile(src, m, fileinfo)
	} else {
		// compare hashes
		destHash, _ := helpers.HashFileMD5(m["dest"])
		if m["md5"] != destHash && !fileinfo.IsDir() {
			// hashes do not match, backup and replace the file
			BackupFile(m["dest"], BACKUP)
			PatchFile(src, m, fileinfo)
		}
	}
}

// Copies from from the patch to the file system
func PatchFile(src string, m map[string]string, fileinfo fs.FileInfo) {
	fmt.Printf("patching %s\n", m["dest"])

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
	// count number of patched files
	num_files += 1
}

// Returns true or false based on what input the user provided to menu prompts
func Confirmed(userInput string) bool {
	c := false
	switch userInput {
	case "y", "Y", "yes", "YES":
		c = true
	case "n", "N", "no", "NO":
		c = false
	default:
		fmt.Printf("\nInvalid option, try again!\n")
	}
	fmt.Printf("\n")
	return c
}

func main() {
	// Display main menu
	for true {
		choice := ""
		switch action := menu(); action {
		case "1":
			// Build a patch
			fmt.Printf("\nProcced with building a patch? ")
			fmt.Scanln(&choice)
			if Confirmed(choice) {
				Build()
			}
		case "2":
			// "Install" the patch
			fmt.Printf("\nProcced with patching? ")
			fmt.Scanln(&choice)
			if Confirmed(choice) {
				Patch()
			}
		case "3":
			// Restore files to pre-patched state
			fmt.Printf("\nProcced with restoring files to their pre-patched state? ")
			fmt.Scanln(&choice)
			if Confirmed(choice) {
				Restore()
			}
		case "4":
			fmt.Printf("\nBye!\n")
			os.Exit(0)
		default:
			fmt.Printf("\nInvalid option, try again!\n\n")
		}
	}
}
