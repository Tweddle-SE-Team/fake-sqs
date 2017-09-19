package main

import (
	"flag"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/Tweddle-SE-Team/goaws/backends/router"
	"github.com/Tweddle-SE-Team/goaws/config"
)

func main() {
	var filename string
	var debug bool
	flag.StringVar(&filename, "config", "", "config file location + name")
	flag.BoolVar(&debug, "debug", true, "debug log level (default Warning)")
	flag.Parse()

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}

	env := "Local"
	if flag.NArg() > 0 {
		env = flag.Arg(0)
	}

	portNumbers := config.LoadYamlConfig(filename, env)

	r := router.New()

	if len(portNumbers) == 1 {
		log.Warnf("SNS and SQS listening on: 0.0.0.0:%s", portNumbers[0])
		err := http.ListenAndServe("0.0.0.0:"+portNumbers[0], r)
		log.Fatal(err)
	} else if len(portNumbers) == 2 {
		go func() {
			log.Warnf("SNS and SQS listening on: 0.0.0.0:%s", portNumbers[0])
			err := http.ListenAndServe("0.0.0.0:"+portNumbers[0], r)
			log.Fatal(err)
		}()
		log.Warnf("SNS and SQS listening on: 0.0.0.0:%s", portNumbers[1])
		err := http.ListenAndServe("0.0.0.0:"+portNumbers[1], r)
		log.Fatal(err)
	} else {
		log.Fatal("Not enough or too many ports defined to start GoAws.")
	}
}
