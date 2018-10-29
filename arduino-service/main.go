package main

import (
	"log"
	"os"
	"path"
	"strings"

	"github.com/jessevdk/go-flags"
	"gopkg.in/src-d/go-git.v4"
)

type gitHash struct {
	Name       string
	LastCommit string
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	var opts struct {
		Boards  []string `short:"b"    long:"board-fqbn"     required:"true"    description:"Arduino FQBN"`
		Ports   []string `short:"p"    long:"port"           required:"true"    description:"USB Port"`
		Sketch  string   `short:"s"    long:"sketch"         required:"true"    description:"Sketch to upload when git-dir changes"`
		LogDir  string   `short:"l"    long:"log-dir"        required:"true"    description:"Directory to store log files"`
		GitDirs []string `short:"g"    long:"git-dir"        required:"true"    description:"Git repo folder for including latest commit hash in payload"`
	}
	_, err := flags.Parse(&opts)
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if _, err := os.Stat(opts.Sketch); os.IsNotExist(err) {
		log.Fatalln(err)
	}

	if stat, err := os.Stat(opts.LogDir); os.IsNotExist(err) {
		log.Fatalln(err)
	} else if !stat.IsDir() {
		log.Fatalf("%s is not a directory", opts.LogDir)
	}

	if len(opts.Boards) > len(opts.Ports) {
		log.Fatalln("Error: more boards than ports")
	} else if len(opts.Ports) > len(opts.Boards) {
		log.Fatalln("Error: more ports than boards")
	}

	for i, b := range opts.Boards {
		logName := path.Join(opts.LogDir, strings.Replace(b, ":", "-", -1)+".log")
		a := NewArduino(b, opts.Ports[i], logName)

		if err := a.Compile(opts.Sketch, gitHashes(opts.GitDirs)); err != nil {
			log.Println("Error: ", err)
		} else {
			if err = a.Upload(opts.Sketch); err != nil {
				log.Println("Error: ", err)
			}
		}

		log.Printf("Starting serial monitor for %s %s\n", opts.Boards[0], opts.Ports[0])
		go a.MonitorSerial()
	}

	select {}
}

func gitHashes(gitDirs []string) []gitHash {
	gitHashes := make([]gitHash, len(gitDirs))
	for i, gitDir := range gitDirs {
		repo, err := git.PlainOpen(gitDir)
		if err != nil {
			log.Fatal("Error: ", err)
		}
		head, err := repo.Head()
		if err != nil {
			log.Fatal("Error: ", err)
		}
		origin, err := repo.Remote("origin")
		if err != nil {
			log.Fatal("Error: ", err)
		}
		repoName := strings.Replace(path.Base(origin.Config().URLs[0]), ".git", "", 1)
		gitHashes[i] = gitHash{repoName, head.Hash().String()[:7]}
		log.Printf("Last commit %v", gitHashes[i])
	}
	return gitHashes
}
