package bot

import (
	"crypto/tls"
	"io"
	"net"
	"strings"
	"time"
	"reflect"
	"errors"
	"fmt"
	"sync"

	"github.com/sorcix/irc"
	rate "github.com/beefsack/go-rate"
)

type Config struct {
	Nickname string
	Username string
	CmdPrefix string
	RateLimitMessages int
	RateLimitDuration time.Duration
	ImageHosts []string
}

type Bot struct {
	Config Config
	Dispatcher
	Auther
	Store
	RateLimiter *RateLimiter
	plugins map[string]*botPlugin
	ic *irc.Conn
}

type Dispatcher interface {
	Event(name string, params ...interface{}) (bool, error)
	Handle(name string, handler Handler)
	RemoveHandler(name string, handler Handler)
}

type trieDispatcher struct {
	tree tNode
}

type tNode struct {
	handlers []Handler
	children map[string]*tNode
}

type Handler func(name string, params []interface{}) (bool, error)

type CmdHandler func(source *irc.Prefix, target string, cmd string, args []string) (bool, error)

type IRCHandler func(msg *irc.Message) (bool, error)

type Permissions interface {
	Can(perm string) bool
}

type PermissionsFunc func(perm string) bool

type Auther interface {
	Auth(mask *irc.Prefix) (Permissions, error)
}

type AuthFunc func(mask *irc.Prefix) (Permissions, error)

type PluginInfo struct {
	Name string
	Description string
}

type Plugin interface {
	Load(bot *Bot) (*PluginInfo, error)
	Unload() error
}

type botPlugin struct {
	*Bot
	plugin Plugin
	Info PluginInfo
	handlerNames []string
	handlers []Handler
}

type RateLimiter struct {
	count int
	duration time.Duration
	limiters map[string]*rate.RateLimiter
	mu sync.Mutex
}

type Store interface {
	Set(key string, value interface{}) error
	Get(key string) (interface{}, error)
	Delete(key string) error
}

type botStore struct {
	bot *Bot
	Store
}

func New(config Config, auth Auther, store Store) *Bot {
	if config.RateLimitMessages <= 0 || config.RateLimitDuration == 0 {
		config.RateLimitMessages = 3
		config.RateLimitDuration = 10 * time.Second
	}
	bot := &Bot{
		Config: config,
		Dispatcher: &trieDispatcher{},
		Auther: auth,
		RateLimiter: NewRateLimiter(config.RateLimitMessages, config.RateLimitDuration),
		plugins: make(map[string]*botPlugin),
	}
	bot.Store = &botStore{bot, store}
	bot.registerHandlers()
	return bot
}

func (b *Bot) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	b.Connect(conn)
	return nil
}

func (b *Bot) DialWithSSL(addr string, config *tls.Config) error {
	conn, err := tls.Dial("tcp", addr, config)
	if err != nil {
		return err
	}
	b.Connect(conn)
	return nil
}

func (b *Bot) Connect(conn net.Conn) {
	b.ic = irc.NewConn(conn)
	go func() {
		b.Event("irc.connect", &irc.Message{})
		for {
			conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
			msg, err := b.ic.Decode()
			if err != nil {
				if err == io.EOF {
					b.Event("irc.disconnect")
				} else if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					b.Event("irc.timeout")
				} else {
					b.Event("irc.error", err)
				}
				return
			}
			_, err = b.Event("irc."+strings.ToLower(msg.Command), msg)
			if err != nil {
				b.Event("irc.error", err)
				return
			}
		}
	}()
}

func (b *Bot) Message(message *irc.Message) error {
	return b.ic.Encode(message)
}

func (node *tNode) Handle(name string, params []interface{}) (bool, error) {
	for _, handler := range node.handlers {
		stop, err := handler(name, params)
		if err != nil {
			return false, err
		}
		if stop {
			return true, nil
		}
	}
	return false, nil
}

func (d *trieDispatcher) Event(name string, params ...interface{}) (bool, error) {
	parts := strings.Split(name, ".")
	node := &d.tree
	for i, part := range parts {
		if wc, ok := node.children["*"]; ok {
			stop, err := wc.Handle(name, params)
			if err != nil {
				return false, err
			}
			if stop {
				return true, nil
			}
		}
		if i == len(parts)-1 {
			if wc, ok := node.children["?"]; ok {
				stop, err := wc.Handle(name, params)
				if err != nil {
					return false, err
				}
				if stop {
					return true, nil
				}
			}
		}
		var ok bool
		node, ok = node.children[part]
		if !ok {
			return false, nil
		}
	}
	stop, err := node.Handle(name, params)
	if err != nil {
		return false, err
	}
	if stop {
		return true, nil
	}
	return false, nil
}

func (d *trieDispatcher) Handle(name string, handler Handler) {
	parts := strings.Split(name, ".")
	node := &d.tree
	for _, part := range parts {
		child, ok := node.children[part]
		if ok {
			node = child
			continue
		}
		child = &tNode{}
		if node.children == nil {
			node.children = make(map[string]*tNode)
		}
		node.children[part] = child
		node = child
	}
	node.handlers = append(node.handlers, handler)
}

func (d *trieDispatcher) RemoveHandler(name string, handler Handler) {
	parts := strings.Split(name, ".")
	parents := make([]*tNode, len(parts))
	node := &d.tree
	for i, part := range parts {
		parents[i] = node
		var ok bool
		node, ok = node.children[part]
		if !ok {
			return
		}
	}
	hp := reflect.ValueOf(handler).Pointer()
	for i, handler := range node.handlers {
		if reflect.ValueOf(handler).Pointer() == hp {
			node.handlers[i] = node.handlers[len(node.handlers)-1]
			node.handlers = node.handlers[:len(node.handlers)-1]
			break
		}
	}
	for i := len(parts) - 1; i >= 0; i -= 1 {
		if len(node.handlers) > 0 || len(node.children) > 0 {
			return
		}
		node = parents[i]
		delete(node.children, parts[i])
	}
}

func (b *Bot) registerHandlers() {
	b.HandleIRC("irc.connect", b.connect)
	b.HandleIRC("irc.ping", b.ping)
	b.HandleIRC("irc.cap", b.caps)
	b.HandleIRC("irc.privmsg", b.cmd)
}

func (b *Bot) connect(msg *irc.Message) (bool, error) {
	messages := []*irc.Message{
		&irc.Message{
			Command: irc.CAP,
			Params: []string{irc.CAP_LS, "302"},
		},
		&irc.Message{
			Command: irc.NICK,
			Params: []string{b.Config.Nickname},
		},
		&irc.Message{
			Command: irc.USER,
			Params: []string{b.Config.Username, "0", "*"},
			Trailing: b.Config.Username,
		},
	}
	for _, msg := range messages {
		err := b.Message(msg)
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func (b *Bot) ping(msg *irc.Message) (bool, error) {
	return true, b.Message(&irc.Message{
		Command: irc.PONG,
		Params: msg.Params,
		Trailing: msg.Trailing,
	})
}

func (b *Bot) caps(msg *irc.Message) (bool, error) {
	switch msg.Params[1] {
	case irc.CAP_LS:
		// FIXME: use theese?
		capabilities := strings.Fields(msg.Trailing)
		err := b.Set("server.capabilities", capabilities)
		if err != nil {
			return false, err
		}
		return false, b.Message(&irc.Message{
			Command: irc.CAP,
			Params: []string{irc.CAP_END},
		})
	}
	return false, nil
}

func (b *Bot) cmd(msg *irc.Message) (bool, error) {
	source := msg.Prefix
	target := msg.Params[0]
	line := msg.Trailing
	if !strings.HasPrefix(line, b.Config.CmdPrefix) {
		return false, nil
	}
	line = line[len(b.Config.CmdPrefix):]
	parts := strings.Fields(line)
	if len(parts) == 0 || len(parts[0]) == 0 {
		return false, nil
	}
	return b.Event("cmd."+parts[0], source, target, parts[0], parts[1:])
}

func (b *Bot) HandleIRC(name string, handler IRCHandler) {
	b.Handle(name, func(name string, params []interface{}) (bool, error) {
		msg := params[0].(*irc.Message)
		return handler(msg)
	})
}

func (b *Bot) HandleCmd(name string, handler CmdHandler) {
	b.Handle(name, func(name string, params []interface{}) (bool, error) {
		source := params[0].(*irc.Prefix)
		target := params[1].(string)
		cmd := params[2].(string)
		args := params[3].([]string)
		return handler(source, target, cmd, args)
	})
}

func (b *Bot) HandleCmdRateLimited(name string, handler CmdHandler) {
	b.Handle(name, func(name string, params []interface{}) (bool, error) {
		source := params[0].(*irc.Prefix)
		target := params[1].(string)
		if b.RateLimiter.Limited(target) {
			return false, nil
		}
		cmd := params[2].(string)
		args := params[3].([]string)
		return handler(source, target, cmd, args)
	})
}

func PrivMsg(target, message string) *irc.Message {
	return &irc.Message{
		Command: irc.PRIVMSG,
		Params: []string{target},
		Trailing: message,
	}
}

func Join(channel string) *irc.Message {
	return &irc.Message{
		Command: irc.JOIN,
		Params: []string{channel},
	}
}

func Ban(channel string, masks ...string) *irc.Message {
	mode := fmt.Sprintf("%s +%s %s", channel, strings.Repeat("b", len(masks)), strings.Join(masks, ""))
	return &irc.Message{
		Command: irc.MODE,
		Params: []string{mode},
	}
}

func Kick(channel string, nick string) *irc.Message {
	return &irc.Message{
		Command: irc.KICK,
		Params: []string{channel, nick},
	}
}

func (pf PermissionsFunc) Can(perm string) bool {
	return pf(perm)
}

func (af AuthFunc) Auth(mask *irc.Prefix) (Permissions, error) {
	return af(mask)
}

func (bp *botPlugin) Handle(name string, handler Handler) {
	bp.handlerNames = append(bp.handlerNames, name)
	bp.handlers = append(bp.handlers, handler)
	bp.Bot.Handle(name, handler)
}

func (bp *botPlugin) Unload() error {
	for i := range bp.handlerNames {
		bp.Bot.RemoveHandler(bp.handlerNames[i], bp.handlers[i])
	}
	return bp.plugin.Unload()
}

func (b *Bot) LoadPlugin(plugin Plugin) error {
	info, err := plugin.Load(b)
	if err != nil {
		return err
	}
	bp := &botPlugin{
		Bot: b,
		plugin: plugin,
		Info: *info,
	}
	b.plugins[info.Name] = bp
	return nil
}

func (b *Bot) UnloadPlugin(name string) error {
	bp, ok := b.plugins[name]
	if !ok {
		return errors.New("Plugin not loaded.")
	}
	return bp.Unload()
}

func NewRateLimiter(count int, duration time.Duration) *RateLimiter {
	return &RateLimiter{
		count: count,
		duration: duration,
		limiters: map[string]*rate.RateLimiter{},
	}
}

func (rl *RateLimiter) Limited(name string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	limiter, ok := rl.limiters[name]
	if !ok {
		limiter = rate.New(rl.count, rl.duration)
		rl.limiters[name] = limiter
	}
	ok, _ = limiter.Try()
	return !ok
}

func (bs *botStore) Get(key string) (interface{}, error) {
	switch key {
	default:
		return bs.Store.Get(key)
	}
}
