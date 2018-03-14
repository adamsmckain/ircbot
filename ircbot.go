package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"time"
	"encoding/json"

	"./bot"
	"./plugins"
	"github.com/sorcix/irc"
	"github.com/turnage/graw/reddit"
)

type ConfigurationAccess struct {
	Mask string
	Access []string
}

type Configuration struct {
    Users map[string][]string
    Nickname string
    Username string
    Password string
    ServerHost string
    ServerPort string
    ServerType string
    SSL bool
    Prefix string
    Channels []string
    ImageHosts []string
}

func main() {
	file, _ := os.Open("config/config.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		log.Fatal(err)
	}
	conf := bot.Config{
		Nickname: configuration.Nickname,
		Username: configuration.Username,
		CmdPrefix: configuration.Prefix,
		ServerType: configuration.ServerType,
		ImageHosts: configuration.ImageHosts,
	}
	auth := bot.AuthFunc(func(mask *irc.Prefix) (bot.Permissions, error) {
		perms, ok := configuration.Users[mask.Host]
		if !ok {
			return nil, nil
		}
		return bot.PermissionsFunc(func(name string) bool {
			for _, perm := range perms {
				if perm == name || perm == "all" {
					return true
				}
			}
			return false
		}), nil
	})
	b := bot.New(conf, auth, make(MapStore))
	b.LoadPlugin(&plugins.Login{Username: configuration.Username, Password: configuration.Password})
	b.LoadPlugin(&plugins.AutoJoin{Channels: configuration.Channels})
	b.LoadPlugin(&plugins.OPCmd{})
	b.LoadPlugin(&plugins.Misc{})
	rs, _ := reddit.NewScript("IRCbot by sizeofcat", 3 * time.Second)
	b.LoadPlugin(&plugins.RedditParser{Lurker: rs})
	b.HandleIRC("irc.*", func(msg *irc.Message) (bool, error) {
		switch strings.ToLower(msg.Command) {
		case "privmsg":
			log.Printf("%s | <%s> %s\n", msg.Params[0], msg.Prefix.Name, msg.Trailing)
		}
		return false, nil
	})
	if (configuration.SSL == true) {
		err = b.DialWithSSL(configuration.ServerHost + ":" + configuration.ServerPort, nil)
	} else {
		err = b.Dial(configuration.ServerHost + ":" + configuration.ServerPort)
	}
	if err != nil {
		log.Fatal(err)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	_ = <-c
}
