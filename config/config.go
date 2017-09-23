package config

import (
	"io/ioutil"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/Tweddle-SE-Team/goaws/services/common"
	"github.com/Tweddle-SE-Team/goaws/services/sns"
	"github.com/Tweddle-SE-Team/goaws/services/sqs"
	"github.com/ghodss/yaml"
)

type EnvSubsciption struct {
	QueueName string
	Raw       bool
}

type EnvTopic struct {
	Name          string
	Subscriptions []EnvSubsciption
}

type EnvQueue struct {
	Name string
}

type Environment struct {
	Host        string
	Port        string
	SqsPort     string
	SnsPort     string
	Region      string
	LogMessages bool
	LogFile     string
	Topics      []EnvTopic
	Queues      []EnvQueue
}

var envs map[string]Environment

func LoadYamlConfig(filename string, env string) []string {
	ports := []string{"4100"}

	if filename == "" {
		filename, _ = filepath.Abs("/etc/goaws/config.yaml")
	}
	log.Warnf("Loading config file: %s", filename)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return ports
	}

	err = yaml.Unmarshal(yamlFile, &envs)
	if err != nil {
		log.Printf("err: %v\n", err)
		return ports
	}
	if env == "" {
		env = "Local"
	}

	if envs[env].Port != "" {
		ports = []string{envs[env].Port}
	} else if envs[env].SqsPort != "" && envs[env].SnsPort != "" {
		ports = []string{envs[env].SqsPort, envs[env].SnsPort}
	}

	common.LogMessages = false
	common.LogFile = "./goaws_messages.log"

	if envs[env].LogMessages == true {
		common.LogMessages = true
		if envs[env].LogFile != "" {
			common.LogFile = envs[env].LogFile
		}
	}
	queueHost := envs[env].Host + ":" + ports[0]
	for _, queueEnv := range envs[env].Queues {
		queue := sqs.NewQueue(queueEnv.Name, queueHost)
		sqs.Service.Queues.Put(queue)
	}
	for _, topic := range envs[env].Topics {
		newTopic := sns.NewTopic(nil, &topic.Name)
		queueEquals := func(s interface{}, v interface{}) bool {
			src := s.(*sqs.Queue)
			value := v.(*string)
			return src.Name == *value
		}
		for _, subs := range topic.Subscriptions {
			q := sqs.Service.Queues.Get(&subs.QueueName, queueEquals)
			if q == nil {
				queue := sqs.NewQueue(subs.QueueName, queueHost)
				sqs.Service.Queues.Put(queue)
				subscription := sns.NewSubscription(newTopic.Arn, "sqs", queue.URL, subs.Raw)
				newTopic.Subscriptions.Put(subscription)
			} else {
				queue := q.(sqs.Queue)
				subscription := sns.NewSubscription(newTopic.Arn, "sqs", queue.URL, subs.Raw)
				newTopic.Subscriptions.Put(subscription)
			}
		}
		sns.Service.Topics.Put(newTopic)
	}
	return ports
}

func GetLogFileName(env string) (string, bool) {
	return envs[env].LogFile, envs[env].LogMessages
}
