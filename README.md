# Patching Utility

This utility creates and deploys patches to files or directories on both Windows and Linux operating systems.

#### Building a Patch

When a patch is built, the specified files or folders are copied to the _files_ directory as well as all parent directories.

1. Open _manifest.cfg_ 

2. Add absolute file paths to all files or directories to be included in the patch. (__one path per line__)

   ```
   /tmp/testing
   ```

3. Run the patch executable and select the _Build_ menu option.

```
Patching Utility Main Menu (linux)
	1. Build
	2. Patch
	3. Restore
	4. Exit
Select an option: 1

Procced with building a patch? y

building...
added /tmp/testing to manifest.
scanning /tmp/testing
added /tmp/testing/folder2 to manifest.
scanning /tmp/testing/folder2
added /tmp/testing/folder2/a.txt to manifest.
creating file backup of /tmp/testing/folder2/a.txt
added /tmp/testing/folder3 to manifest.
scanning /tmp/testing/folder3
added /tmp/testing/folder3/a.txt to manifest.
creating file backup of /tmp/testing/folder3/a.txt
added /tmp/testing/t.txt to manifest.
creating file backup of /tmp/testing/t.txt
build complete. added 6 files
```

#### Patching

Files are patched based on MD5 checksum comparisons between the patched file and the file that is present on the file system, or when a file is missing. The _manifest_ files contains hash values, destination file paths, and a few additional properties when using the Linux OS (Uid, Gid, permissions).

```
Patching Utility Main Menu (linux)
	1. Build
	2. Patch
	3. Restore
	4. Exit
Select an option: 2

Procced with patching? y

patching...
creating file backup of /tmp/testing/folder2/a.txt
patching /tmp/testing/folder2/a.txt
complete. patched 1 files
```

#### Restoring

Before a file is patched, a backup of the file to be replaced is taken and stored in the patches _build_ directory. Files are restored when hash values of the patched file differ from the hash value of the backed up file.

```
Patching Utility Main Menu (linux)
	1. Build
	2. Patch
	3. Restore
	4. Exit
Select an option: 3

Procced with restoring files to their pre-patched state? y

restoring...
restoring /tmp/testing/folder2/a.txt
complete. restored 1 files
```



## Building

#### Linux

```bash
go build -o patch-cli main.go
```

#### Windows (64-bit)

```bash
env GOOS=windows GOARCH=amd64 go build -o patch-cli.exe main.go
```