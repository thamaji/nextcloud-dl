nextcloud-dl
====

### Usage

```
$ nextcloud-dl -h

Usage: nextcloud-dl [OPTIONS] URL [URL...]

Download files from Nextcloud

Options:
  -h	show help
  -o string
    	set output directory (default ".")
  -p string
    	set password
  -u string
    	set username
  -v	show version

```

### Example

```
$ nextcloud-dl https://path-to-nextcloud/index.php/apps/files?dir=//Photos
Enter username: myusername
Enter password: mypassword
/Photos/Coast.jpg
/Photos/Hummingbird.jpg
/Photos/Nut.jpg

$ ls Photos
Coast.jpg  Hummingbird.jpg  Nut.jpg
```
