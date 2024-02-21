package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func goDotEnvVariable(key string) string {

	err := godotenv.Load("sftp_config.txt")

	if err != nil {
		log.Fatal("Error loading the config file")
	}

	return os.Getenv(key)
}

func main() {

	host := goDotEnvVariable("HOST_URL")
	user := goDotEnvVariable("USERNAME")
	pass := goDotEnvVariable("PASSWORD")
	port := ":" + goDotEnvVariable("PORT")
    localDir := goDotEnvVariable("LOCAL_DIRECTORY_TO_SAVE")
    fileOrDirectoryToFind := goDotEnvVariable("FILE_OR_DIR_TO_DOWNLOAD")

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		// this shouldnt be used
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", host+port, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(1)
	}

	workingDirectory, err := client.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fail to get the current working directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Working Directory: %s\n", workingDirectory)

    remoteDir := filepath.Join(workingDirectory, fileOrDirectoryToFind)

    fileInfo, err := client.Stat(remoteDir)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to get file info: %v\n", err)
        os.Exit(1)
    }

        if fileInfo.IsDir() {
        err = downloadDirectory(client, remoteDir, localDir)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Failed to download directory: %v\n", err)
            os.Exit(1)
        }
    } else {
        err = downloadFile(client, remoteDir, localDir)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Failed to download file: %v\n", err)
            os.Exit(1)
        }
    }

}

func downloadFile(client *sftp.Client, remoteFile, localFile string) error {
	remote, err := client.Open(remoteFile)
	if err != nil {
		return err
	}
	defer remote.Close()

	local, err := os.Create(localFile)
	if err != nil {
		return err
	}
	defer local.Close()

	_, err = io.Copy(local, remote)
	return err
}

func downloadDirectory(client *sftp.Client, remoteDir, localDir string) error {
	entries, err := client.ReadDir(remoteDir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(localDir, os.ModePerm)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		remotePath := filepath.Join(remoteDir, entry.Name())
		localPath := filepath.Join(localDir, entry.Name())

		if entry.IsDir() {
			err = downloadDirectory(client, remotePath, localPath)
			if err != nil {
				return err
			}
		} else {
			err = downloadFile(client, remotePath, localPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
