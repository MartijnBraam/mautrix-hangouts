// mautrix-whatsapp - A Matrix-WhatsApp puppeting bridge.
// Copyright (C) 2019 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	flag "maunium.net/go/mauflag"
	log "maunium.net/go/maulogger/v2"
	"maunium.net/go/mautrix-appservice"

	"mautrix-hangouts/config"
	"mautrix-hangouts/database"
	"mautrix-hangouts/types"
)

var configPath = flag.MakeFull("c", "config", "The path to your config file.", "config.yaml").String()

//var baseConfigPath = flag.MakeFull("b", "base-config", "The path to the example config file.", "example-config.yaml").String()
var registrationPath = flag.MakeFull("r", "registration", "The path where to save the appservice registration.", "registration.yaml").String()
var generateRegistration = flag.MakeFull("g", "generate-registration", "Generate registration and quit.", "false").Bool()
var wantHelp, _ = flag.MakeHelpFlag()

func (bridge *Bridge) GenerateRegistration() {
	reg, err := bridge.Config.NewRegistration()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to generate registration:", err)
		os.Exit(20)
	}

	err = reg.Save(*registrationPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save registration:", err)
		os.Exit(21)
	}

	err = bridge.Config.Save(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save config:", err)
		os.Exit(22)
	}
	fmt.Println("Registration generated. Add the path to the registration to your Synapse config, restart it, then start the bridge.")
	os.Exit(0)
}

type Bridge struct {
	AS             *appservice.AppService
	EventProcessor *appservice.EventProcessor
	MatrixHandler  *MatrixHandler
	Config         *config.Config
	DB             *database.Database
	Log            log.Logger
	StateStore     *AutosavingStateStore
	Bot            *appservice.IntentAPI
	Formatter      *Formatter

	usersByMXID         map[types.MatrixUserId]*User
	UsersByHID          map[types.HangoutsId]*User
	usersLock           sync.Mutex
	managementRooms     map[types.MatrixRoomId]*User
	managementRoomsLock sync.Mutex
	portalsByMXID       map[types.MatrixRoomId]*Portal
	portalsByHID        map[database.PortalKey]*Portal
	portalsLock         sync.Mutex
	puppets             map[types.HangoutsId]*Puppet
	puppetsByCustomMXID map[types.MatrixUserId]*Puppet
	puppetsLock         sync.Mutex
}

func NewBridge() *Bridge {
	bridge := &Bridge{
		usersByMXID:         make(map[types.MatrixUserId]*User),
		UsersByHID:          make(map[types.HangoutsId]*User),
		managementRooms:     make(map[types.MatrixRoomId]*User),
		portalsByMXID:       make(map[types.MatrixRoomId]*Portal),
		portalsByHID:        make(map[database.PortalKey]*Portal),
		puppets:             make(map[types.HangoutsId]*Puppet),
		puppetsByCustomMXID: make(map[types.MatrixUserId]*Puppet),
	}

	var err error
	bridge.Config, err = config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(10)
	}
	return bridge
}

func (bridge *Bridge) Init() {
	var err error

	bridge.AS, err = bridge.Config.MakeAppService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to initialize AppService:", err)
		os.Exit(11)
	}
	bridge.AS.Init()
	bridge.Bot = bridge.AS.BotIntent()

	bridge.Log = log.Create()
	bridge.Config.Logging.Configure(bridge.Log)
	log.DefaultLogger = bridge.Log.(*log.BasicLogger)
	err = log.OpenFile()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to open log file:", err)
		os.Exit(12)
	}
	bridge.AS.Log = log.Sub("Matrix")

	bridge.Log.Debugln("Initializing state store")
	bridge.StateStore = NewAutosavingStateStore(bridge.Config.AppService.StateStore)
	err = bridge.StateStore.Load()
	if err != nil {
		bridge.Log.Fatalln("Failed to load state store:", err)
		os.Exit(13)
	}
	bridge.AS.StateStore = bridge.StateStore

	bridge.Log.Debugln("Initializing database")
	bridge.DB, err = database.New(bridge.Config.AppService.Database.Type, bridge.Config.AppService.Database.URI)
	if err != nil {
		bridge.Log.Fatalln("Failed to initialize database:", err)
		os.Exit(14)
	}

	bridge.DB.SetMaxOpenConns(bridge.Config.AppService.Database.MaxOpenConns)
	bridge.DB.SetMaxIdleConns(bridge.Config.AppService.Database.MaxIdleConns)

	bridge.Log.Debugln("Initializing Matrix event processor")
	bridge.EventProcessor = appservice.NewEventProcessor(bridge.AS)
	bridge.Log.Debugln("Initializing Matrix event handler")
	bridge.MatrixHandler = NewMatrixHandler(bridge)
	bridge.Formatter = NewFormatter(bridge)
}

func (bridge *Bridge) Start() {
	err := bridge.DB.Init(bridge.Config.AppService.Database.Type)
	if err != nil {
		bridge.Log.Fatalln("Failed to initialize database:", err)
		os.Exit(15)
	}
	bridge.Log.Debugln("Starting application service HTTP server")
	go bridge.AS.Start()
	bridge.Log.Debugln("Starting event processor")
	go bridge.EventProcessor.Start()
	go bridge.UpdateBotProfile()
	go bridge.StartUsers()
}

func (bridge *Bridge) UpdateBotProfile() {
	bridge.Log.Debugln("Updating bot profile")
	botConfig := bridge.Config.AppService.Bot

	var err error
	if botConfig.Avatar == "remove" {
		err = bridge.Bot.SetAvatarURL("")
	} else if len(botConfig.Avatar) > 0 {
		err = bridge.Bot.SetAvatarURL(botConfig.Avatar)
	}
	if err != nil {
		bridge.Log.Warnln("Failed to update bot avatar:", err)
	}

	if botConfig.Displayname == "remove" {
		err = bridge.Bot.SetDisplayName("")
	} else if len(botConfig.Avatar) > 0 {
		err = bridge.Bot.SetDisplayName(botConfig.Displayname)
	}
	if err != nil {
		bridge.Log.Warnln("Failed to update bot displayname:", err)
	}
}

func (bridge *Bridge) StartUsers() {
	bridge.Log.Debugln("Starting users")
	for _, user := range bridge.GetAllUsers() {
		go user.Connect(false)
	}
	bridge.Log.Debugln("Starting custom puppets")
	for _, puppet := range bridge.GetAllPuppetsWithCustomMXID() {
		go func() {
			puppet.log.Debugln("Starting custom puppet", puppet.CustomMXID)
			err := puppet.StartCustomMXID()
			if err != nil {
				puppet.log.Errorln("Failed to start custom puppet:", err)
			}
		}()
	}
}

func (bridge *Bridge) Stop() {
	bridge.AS.Stop()
	bridge.EventProcessor.Stop()
	for _, user := range bridge.UsersByHID {
		if user.Conn == nil {
			continue
		}
		bridge.Log.Debugln("Disconnecting", user.MXID)
		sess, err := user.Conn.Disconnect()
		if err != nil {
			bridge.Log.Errorfln("Error while disconnecting %s: %v", user.MXID, err)
		} else {
			user.SetSession(&sess)
		}
	}
	err := bridge.StateStore.Save()
	if err != nil {
		bridge.Log.Warnln("Failed to save state store:", err)
	}
}

func (bridge *Bridge) Main() {
	if *generateRegistration {
		bridge.GenerateRegistration()
		return
	}

	bridge.Init()
	bridge.Log.Infoln("Bridge initialization complete, starting...")
	bridge.Start()
	bridge.Log.Infoln("Bridge started!")

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	bridge.Log.Infoln("Interrupt received, stopping...")
	bridge.Stop()
	bridge.Log.Infoln("Bridge stopped.")
	os.Exit(0)
}

func main() {
	flag.SetHelpTitles(
		"mautrix-whatsapp - A Matrix-WhatsApp puppeting bridge.",
		"mautrix-whatsapp [-h] [-c <path>] [-r <path>] [-g]")
	err := flag.Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.PrintHelp()
		os.Exit(1)
	} else if *wantHelp {
		flag.PrintHelp()
		os.Exit(0)
	}

	NewBridge().Main()
}
