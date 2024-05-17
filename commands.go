package main

import (
	"strings"

	"maunium.net/go/mautrix/bridge/commands"
)

type WrappedCommandEvent struct {
	*commands.Event
	Bridge *IMAPBridge
	User   *User
	Portal *Portal
}

func (br *IMAPBridge) RegisterCommands() {
	proc := br.CommandProcessor.(*commands.Processor)
	proc.AddHandlers(
		cmdPing,
		cmdLogin,
	)
}

func wrapCommand(handler func(*WrappedCommandEvent)) func(*commands.Event) {
	return func(ce *commands.Event) {
		user := ce.User.(*User)
		var portal *Portal
		if ce.Portal != nil {
			portal = ce.Portal.(*Portal)
		}
		br := ce.Bridge.Child.(*IMAPBridge)
		handler(&WrappedCommandEvent{ce, br, user, portal})
	}
}

var cmdLogin = &commands.FullHandler{
	Func: wrapCommand(fnLogin),
	Name: "login",
	Help: commands.HelpMeta{
		Section:     commands.HelpSectionAuth,
		Description: "Link the bridge to your email account.",
	},
}

func fnLogin(ce *WrappedCommandEvent) {
	if len(ce.Args) < 2 {
		ce.Reply("**Usage**: $cmdprefix login <email> <password>")
		return
	}

	if ce.User.Client != nil && ce.User.Client.IsLoggedIn() {
		ce.Reply("%s is already logged %s", ce.Args[0], ce.Args[1])
		return
	}

	user := ce.Bridge.GetUserByMXID(ce.User.MXID)
	reply, err := user.Login(ce.Ctx, ce.Args[0], strings.Join(ce.Args[1:], " "))
	if err != nil {
		ce.Reply(reply)
		return
	}

	ce.Reply("Successfully logged")
}

var cmdPing = &commands.FullHandler{
	Func: wrapCommand(fnPing),
	Name: "ping",
	Help: commands.HelpMeta{
		Section:     commands.HelpSectionAuth,
		Description: "Check your connection to IMAP",
	},
}

func fnPing(ce *WrappedCommandEvent) {
	if ce.User.EmailAddress == "" {
		ce.Reply("You're not logged in")
	} else if !ce.User.IsLoggedIn() {
		ce.Reply("You were logged in at some point, but are not anymore")
	} else {
		ce.Reply("You're logged in")
	}
}
