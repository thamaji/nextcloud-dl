package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/studio-b12/gowebdav"
	"golang.org/x/crypto/ssh/terminal"
)

func usage() {
	output := flag.CommandLine.Output()
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Usage: "+os.Args[0]+" [OPTIONS] URL [URL...]")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Download files from Nextcloud")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Options:")
	flag.CommandLine.PrintDefaults()
}

func walk(client *gowebdav.Client, filePath string, f func(string, os.FileInfo) error) error {
	stat, err := client.Stat(filePath)
	if err != nil {
		return err
	}

	return _walk(client, filePath, stat, f)
}

func _walk(client *gowebdav.Client, filePath string, stat os.FileInfo, f func(string, os.FileInfo) error) error {
	if !stat.IsDir() {
		return f(filePath, stat)
	}

	files, err := client.ReadDir(filePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := _walk(client, gowebdav.Join(filePath, file.Name()), file, f); err != nil {
			return err
		}
	}

	return nil
}

func parseURL(rawURL string) (string, string, error) {
	url, err := url.Parse(rawURL)
	if err != nil {
		return "", "", err
	}

	i := strings.Index(url.Path, "/index.php/apps/files")
	if i < 0 {
		i = strings.Index(url.Path, "/apps/files")
	}
	if i >= 0 {
		filePath := path.Clean(url.Query().Get("dir"))
		url.RawQuery = ""
		url.Path = path.Join(url.Path[:i], "/remote.php/webdav")
		return url.String(), filePath, nil
	}

	if i := strings.Index(url.Path, "/remote.php/webdav"); i > 0 {
		filePath := url.Path[i+len("/remote.php/webdav"):]
		url.RawQuery = ""
		url.Path = url.Path[:i+len("/remote.php/webdav")]
		return url.String(), filePath, nil
	}

	return "", "", errors.New("unsupported url: " + rawURL)
}

func download(client *gowebdav.Client, remotePath string, stat os.FileInfo, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	localFile, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		err1 := localFile.Close()
		if err == nil {
			err = err1
		}
	}()

	fmt.Println(remotePath)

	remoteFile, err := client.ReadStream(remotePath)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(ioutil.Discard, remoteFile)
		remoteFile.Close()
	}()

	_, err = io.Copy(localFile, remoteFile)

	return err
}

func readUsername(stdin int) (string, error) {
	if !terminal.IsTerminal(stdin) {
		return "", errors.New("stdin is not a terminal")
	}

	var username string
	fmt.Print("Enter username: ")
	_, err := fmt.Fscanln(os.Stdin, &username)
	return username, err
}

func readPassword(stdin int) (string, error) {
	if !terminal.IsTerminal(stdin) {
		return "", errors.New("stdin is not a terminal")
	}

	fmt.Print("Enter password: ")
	bytes, err := terminal.ReadPassword(stdin)
	fmt.Println()
	return string(bytes), err
}

func main() {
	flag.Usage = usage

	var username, password, directory string
	var version, help bool

	var err error

	flag.StringVar(&username, "u", "", "set username")
	flag.StringVar(&password, "p", "", "set password")
	flag.StringVar(&directory, "o", ".", "set output directory")

	flag.BoolVar(&version, "v", false, "show version")
	flag.BoolVar(&help, "h", false, "show help")

	flag.Parse()

	if help {
		usage()
		return
	}

	if version {
		fmt.Fprintln(os.Stdout, "1.0.0")
		return
	}

	args := flag.Args()

	if len(args) < 1 {
		usage()
		return
	}

	stdin := int(os.Stdin.Fd())

	if username == "" {
		username, err = readUsername(stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}
	}

	if password == "" {
		password, err = readPassword(stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}
	}

	for _, url := range args {
		webdavURL, root, err := parseURL(url)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}

		client := gowebdav.NewClient(webdavURL, username, password)
		client.Connect()

		err = walk(client, root, func(remotePath string, stat os.FileInfo) error {
			if root == "." || root == "/" {
				return download(client, remotePath, stat, filepath.Join(directory, username, filepath.FromSlash(remotePath)))
			}

			relPath, err := filepath.Rel(filepath.FromSlash(path.Dir(root)), filepath.FromSlash(remotePath))
			if err != nil {
				return err
			}

			if relPath == "." {
				return download(client, remotePath, stat, filepath.Join(directory, filepath.Base(filepath.FromSlash(remotePath))))
			}

			return download(client, remotePath, stat, filepath.Join(directory, relPath))
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}
	}
}
