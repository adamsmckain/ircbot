package plugins

import (
	"os"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
	"log"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"../bot"
	"github.com/sorcix/irc"
	"github.com/turnage/graw/reddit"
)

type AutoJoin struct {
	Channels []string
	bot *bot.Bot
}

type Misc struct {
	bot *bot.Bot
	bannedUsers map[string]string
}

type OPCmd struct {
	bot *bot.Bot
}

type LoginX struct {
	Username string
	Password string
	bot *bot.Bot
}

type RedditParser struct {
	PreloadCount int
	Lurker reddit.Lurker
	bot *bot.Bot
	close chan bool
}

type RedditSearch struct {
	Commands []string
	Subreddits []string
	RedditListTag string
	What []string
	Check func(*reddit.Post, *bot.Bot) bool
	posts []*reddit.Post
	mu sync.Mutex
	close chan bool
	NSFW bool
}

type ReplyTerms map[string]string

var RedditSearches = []RedditSearch{
	RedditSearch{
		Commands: []string{"nsfw"},
		Subreddits: []string{
			"nsfw", "nsfwhardcore", "nsfw2", "HighResNSFW", "BonerMaterial",
			"porn", "iWantToFuckHer", "NSFW_nospam", "Sexy", "nude",
			"UnrealGirls", "primes", "THEGOLDSTANDARD", "nsfw_hd", "UHDnsfw",
			"BeautifulTitsAndAss", "FuckMarryOrKill", "NSFWCute",
			"badassgirls", "HotGirls", "PornPleasure", "nsfwnonporn",
			"NSFWcringe", "NSFW_PORN_ONLY", "Sex_Games", "BareGirls",
			"lusciousladies", "Babes", "FilthyGirls", "NaturalWomen",
			"ImgurNSFW", "Adultpics", "sexynsfw", "nsfw_sets", "OnlyGoodPorn",
			"TumblrArchives", "HardcoreSex", "PornLovers", "NSFWgaming",
			"Fapucational", "RealBeauties", "fappitt", "exotic_oasis", "TIFT",
			"nakedbabes", "oculusnsfw", "CrossEyedFap", "TitsAssandNoClass",
			"formylover", "Ass_and_Titties", "Ranked_Girls", "fapfactory",
			"NSFW_hardcore", "Sexyness", "debs_and_doxies", "nsfwonly",
			"pornpedia", "lineups", "Nightlysex", "spod", "nsfwnew",
			"pinupstyle", "NoBSNSFW", "nsfwdumps", "FoxyLadies",
			"nsfwcloseups", "NudeBeauty", "SimplyNaked", "fappygood",
			"FaptasticImages", "WhichOneWouldYouPick", "TumblrPorn",
			"SaturdayMorningGirls", "NSFWSector", "GirlsWithBigGuns",
			"QualityNsfw", "nsfwPhotoshopBattles", "hawtness",
			"fapb4momgetshome", "SeaSquared", "SexyButNotPorn", "WoahPoon",
			"Reflections", "Hotness", "Erotic_Galleries", "carnalclass",
			"nsfw_bw", "LaBeauteFeminine", "Sweet_Sexuality", "NSFWart",
			"WomenOfColorRisque",
		},
		What: []string{"nsfw"},
		Check: checkIsImage,
		NSFW: true,
	},
	RedditSearch{
		Commands: []string{"ass"},
		Subreddits: []string{
			"AssOnTheGlass", "BoltedOnBooty", "BubbleButts",
			"ButtsAndBareFeet", "Cheeking", "HighResASS",
			"LoveToWatchYouLeave", "NoTorso", "SpreadEm", "TheUnderbun",
			"Top_Tier_Asses", "Tushy", "Underbun", "ass", "assgifs",
			"bigasses", "booty", "booty_gifs", "datass", "datbuttfromthefront",
			"hugeass", "juicybooty", "pawg", "twerking", "whooties",
		},
		What: []string{"ass"},
		Check: checkIsImage,
		NSFW: true,
	},
	RedditSearch{
		Commands: []string{"boobs"},
		Subreddits: []string{
			"BeforeAndAfterBoltons", "Bigtitssmalltits", "BoltedOnMaxed",
			"Boobies", "BreastEnvy", "EpicCleavage", "HardBoltOns",
			"JustOneBoob", "OneInOneOut", "PM_ME_YOUR_TITS_GIRL",
			"PerfectTits", "Perky", "Rush_Boobs", "Saggy", "SloMoBoobs",
			"TheHangingBoobs", "TheUnderboob", "Titsgalore", "TittyDrop",
			"bananatits", "boltedontits", "boobbounce", "boobgifs",
			"boobkarma", "boobland", "boobs", "breastplay", "breasts",
			"cleavage", "feelthemup", "handbra", "hanging", "hersheyskisstits",
			"homegrowntits", "knockers", "naturaltitties", "sideboob",
			"tits", "titsagainstglass", "torpedotits", "underboob",
		},
		What: []string{"boobs", "tits", "titties"},
		Check: checkIsImage,
		NSFW: true,
	},
	RedditSearch{
		Commands: []string{"trap"},
		Subreddits: []string{
			"Ladyboys", "asianladyboy", "transgif", "dickgirls", "futanari",
		},
		What: []string{"trap"},
		Check: checkIsImage,
		NSFW: true,
	},
	RedditSearch{
		Commands: []string{"milf"},
		Subreddits: []string{
			"milf",
		},
		What: []string{"milf"},
		Check: checkIsImage,
		NSFW: true,
	},
	RedditSearch{
		Commands: []string{"redhead"},
		Subreddits: []string{
			"redheads", "ginger", "redhead",
		},
		What: []string{"redhead", "ginger"},
		Check: checkIsImage,
		NSFW: true,
	},
	RedditSearch{
		Commands: []string{"cat"},
		Subreddits: []string{
			"cat", "cats", "CatGifs", "KittenGifs", "Kittens", "CatPics",
			"Kitties", "Kitty", "CatPictures", "LookAtMyCat", "CatReddit", 
			"CatSpotting", "Kitten", "DelightfullyChubby",
		},
		What: []string{"cat", "kitten", "kitty"},
		Check: checkIsImage,
		NSFW: false,
	},
	RedditSearch{
		Commands: []string{"dog"},
		Subreddits: []string{
			"dog", "dogs", "lookatmydog", "DogPictures", "dogswearinghats",
			"dogswatchingyoueat",
		},
		What: []string{"dog", "puppy", "puppeh"},
		Check: checkIsImage,
		NSFW: false,
	},
	RedditSearch{
		Commands: []string{"blonde"},
		Subreddits: []string{
			"blonde", "blondes",
		},
		What: []string{"blonde"},
		Check: checkIsImage,
		NSFW: true,
	},
	RedditSearch{
		Commands: []string{"brunette"},
		Subreddits: []string{
			"brunette", "brunetteass",
		},
		What: []string{"brunette"},
		Check: checkIsImage,
		NSFW: true,
	},
}

func (p *AutoJoin) Load(b *bot.Bot) (*bot.PluginInfo, error) {
	p.bot = b
	b.Handle("irc.376", p.welcome)
	return &bot.PluginInfo{
		Name: "AutoJoin",
		Description: "Auto joins channels upon connect.",
	}, nil
}

func (p *AutoJoin) welcome(name string, params []interface{}) (bool, error) {
	for _, channel := range p.Channels {
		err := p.bot.Message(bot.Join(channel))
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func (p *AutoJoin) Unload() error {
	return nil
}

func (p *Misc) Load(b *bot.Bot) (*bot.PluginInfo, error) {
	p.bot = b
	p.textCmd("cmd.hey", []string{"how are you?", "heya!", "hello"})
	p.bot.HandleCmdRateLimited("cmd.buzz", p.buzz)
	file, _ := os.Open("config/reply.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	replyTerms := ReplyTerms{}
	err := decoder.Decode(&replyTerms)
	if err != nil {
		log.Fatal(err)
	}
	for key, value := range replyTerms {
		value := value
		key := key
		p.textReply("irc.privmsg", value, func(line string) bool {
			line = strings.ToLower(line)
			return strings.HasSuffix(line, key)
		})
	}
	p.bot.HandleIRC("irc.invite", p.invite)
	p.bot.HandleIRC("irc.kick", p.kick)
	p.bot.HandleIRC("irc.join", p.join)
	p.bot.HandleCmdRateLimited("cmd.bs", p.bullshit)
	p.bannedUsers = make(map[string]string)
	return &bot.PluginInfo{
		Name: "Misc",
		Description: "Miscellaneous commands.",
	}, nil
}

func (p *Misc) Unload() error {
	return nil
}

func (p *Misc) textCmd(cmd string, texts []string) {
	if len(texts) == 0 {
		return
	}
	handler := func(source *irc.Prefix, target string, cmd string, args []string) (bool, error) {
		text := texts[rand.Intn(len(texts))]
		if len(args) > 0 {
			text = args[0] + ": " + text
		}
		p.bot.Message(bot.PrivMsg(target, text))
		return true, nil
	}
	p.bot.HandleCmdRateLimited(cmd, handler)
}

func (p *Misc) textReply(cmd, text string, check func(string) bool) {
	handler := func(msg *irc.Message) (bool, error) {
		if !check(msg.Trailing) {
			return false, nil
		}
		if p.bot.RateLimiter.Limited(msg.Params[0]) {
			return false, nil
		}
		p.bot.Message(bot.PrivMsg(msg.Params[0], text))
		return false, nil
	}
	p.bot.HandleIRC(cmd, handler)
}

func (p *Misc) bullshit(source *irc.Prefix, target string, cmd string, args []string) (bool, error) {
	if len(args) == 0 {
		return true, nil
	}
	msg := fmt.Sprintf("%s: I am bullshitting you!", args[0])
	p.bot.Message(bot.PrivMsg(target, msg))
	return true, nil
}

func (p *Misc) buzz(source *irc.Prefix, target string, cmd string, args []string) (bool, error) {
	if len(args) == 0 {
		return true, nil
	}
	perms, err := p.bot.Auth(source)
	if err != nil {
		return false, err
	}
	if perms == nil || !perms.Can("annoy") {
		return true, nil
	}
	lines := []string{
		"%s",
		"%s!",
		"paging %s!",
		"BUZZING %s",
		"%s %[1]s %[1]s %[1]s %[1]s",
		"hey %s!",
		"%s %[1]s %[1]s %[1]s",
		"%s come on",
	}
	times := rand.Intn(3) + 3
	for i := 0; i < times; i++ {
		line := lines[rand.Intn(len(lines))]
		msg := fmt.Sprintf(line, args[0])
		p.bot.Message(bot.PrivMsg(target, msg))
		time.Sleep(time.Duration(rand.Intn(300) + 300) * time.Millisecond)
	}
	return true, nil
}

func (p *Misc) invite(msg *irc.Message) (bool, error) {
	perms, err := p.bot.Auth(msg.Prefix)
	if err != nil {
		return false, err
	}
	if perms == nil || !perms.Can("invite") {
		return true, nil
	}
	//channel := msg.Trailing
	channel := msg.Params[1]
	err = p.bot.Message(bot.Join(channel))
	return true, nil
}

func (p *Misc) kick(msg *irc.Message) (bool, error) {
	channel, who := msg.Params[0], msg.Params[1]
	if who != p.bot.Config.Nickname {
		return false, nil
	}
	bannedUser := msg.Prefix.Name
	if bannedUser == "X" {
		parts := strings.Fields(msg.Trailing)
		bannedUser = strings.Trim(parts[len(parts) - 1], "()")
	}
	p.bannedUsers[channel] = bannedUser
	return false, nil
}

func (p *Misc) join(msg *irc.Message) (bool, error) {
	if msg.Prefix.Name != p.bot.Config.Nickname {
		return false, nil
	}
	channel := msg.Trailing
	bannedUser, ok := p.bannedUsers[channel]
	if !ok {
		return false, nil
	}
	delete(p.bannedUsers, bannedUser)
	welcome := fmt.Sprintf("%s: _)_", bannedUser)
	p.bot.Message(bot.PrivMsg(channel, welcome))
	return false, nil
}

func (p *OPCmd) Load(b *bot.Bot) (*bot.PluginInfo, error) {
	p.bot = b
	b.HandleCmd("cmd.kb", p.kickban)
	return &bot.PluginInfo{
		Name: "OPCmd",
		Description: "OP Commands.",
	}, nil
}

func (p *OPCmd) Unload() error {
	return nil
}

func (p *OPCmd) kickban(source *irc.Prefix, target string, cmd string, args []string) (bool, error) {
	if len(args) != 1 {
		return true, nil
	}
	perms, err := p.bot.Auth(source)
	if err != nil {
		return false, err
	}
	if perms == nil || !perms.Can("opcmds") {
		return true, nil
	}
	whom := args[0]
	p.bot.Message(bot.PrivMsg("X", fmt.Sprintf("ban %s %s", target, whom)))
	return true, nil
}

func (p *LoginX) Load(b *bot.Bot) (*bot.PluginInfo, error) {
	p.bot = b
	b.Handle("irc.001", p.welcome)
	return &bot.PluginInfo{
		Name: "LoginX",
		Description: "Authenticate to X.",
	}, nil
}

func (p *LoginX) Unload() error {
	return nil
}

func (p *LoginX) welcome(name string, params []interface{}) (bool, error) {
	if len(p.Username) > 0 && len(p.Password) > 0 {
		p.bot.Message(bot.PrivMsg("x@channels.undernet.org", "login " + p.Username + " " + p.Password))
	}
	return false, nil
}

func (p *RedditParser) Load(b *bot.Bot) (*bot.PluginInfo, error) {
	p.bot = b
	p.close = make(chan bool)
	if p.PreloadCount < 1 {
		p.PreloadCount = 10
	}
	for i := range RedditSearches {
		RedditSearches[i].register(p)
	}
	p.bot.HandleCmdRateLimited("cmd.porn", p.roulette)
	return &bot.PluginInfo{
		Name: "RedditParser",
		Description: "Parse Reddit for useful images.",
	}, nil
}

func (p *RedditParser) Unload() error {
	close(p.close)
	return nil
}

func (p *RedditParser) roulette(source *irc.Prefix, target string, cmd string, args []string) (bool, error) {
	RedditSearch := RedditSearches[rand.Intn(len(RedditSearches))]
	cmd = RedditSearch.Commands[0]
	return p.bot.Event("cmd." + cmd, source, target, cmd, args)
}

func checkIsImage(post *reddit.Post, b *bot.Bot) bool {
	linkURL, err := url.Parse(post.URL)
	if err != nil {
		return false
	}
	for _, host := range b.Config.ImageHosts {
		if strings.Contains(linkURL.Host, host) {
			return true
		}
	}
	return false
}

func chooseRandStr(opt []string) string {
	return opt[rand.Intn(len(opt))]
}

func (m *RedditSearch) get() *reddit.Post {
	for i := 0; i < 5; i++ {
		m.mu.Lock()
		var post *reddit.Post
		if len(m.posts) > 0 {
			post = m.posts[len(m.posts) - 1]
			m.posts = m.posts[:len(m.posts) - 1]
		}
		m.mu.Unlock()
		if post != nil {
			return post
		}
		select {
		case <-m.close:
			return nil
		case <-time.After(time.Second):
		}
	}
	return nil
}

func (m *RedditSearch) register(plug *RedditParser) {
	m.posts = make([]*reddit.Post, 0, plug.PreloadCount)
	m.close = plug.close
	go func() {
		if len(m.RedditListTag) > 0 {
			m.getSubredditList()
		}
		if len(m.Subreddits) == 0 {
			return
		}
		m.preload(plug.Lurker, plug.bot)
	}()
	handler := func(source *irc.Prefix, target string, cmd string, args []string) (bool, error) {
		what := chooseRandStr(m.What)
		post := m.get()
		if post == nil {
			plug.bot.Message(bot.PrivMsg(target, fmt.Sprintf("%s: haven't indexed any %s yet", source.Name, what)))
			return true, nil
		}
		var msg string
		if len(args) > 0 {
			if m.NSFW == true {
				msg = fmt.Sprintf("%s, here is some %s from %s: %s NSFW (https://redd.it/%s)", args[0], what, source.Name, post.URL, post.ID)
			} else {
				msg = fmt.Sprintf("%s, here is some %s from %s: %s (https://redd.it/%s)", args[0], what, source.Name, post.URL, post.ID)
			}
		} else {
			if m.NSFW == true {
				msg = fmt.Sprintf("%s, here is some %s: %s NSFW (https://redd.it/%s)", source.Name, what, post.URL, post.ID)
			} else {
				msg = fmt.Sprintf("%s, here is some %s: %s (https://redd.it/%s)", source.Name, what, post.URL, post.ID)
			}
		}
		plug.bot.Message(bot.PrivMsg(target, msg))
		return true, nil
	}
	for _, cmd := range m.Commands {
		plug.bot.HandleCmdRateLimited("cmd." + cmd, handler)
	}
}

func (m *RedditSearch) getSubredditList() {
	var url string
	if m.NSFW == true {
		url = "http://redditlist.com/nsfw/category/" + m.RedditListTag
	} else {
		url = "http://redditlist.com/sfw/category/" + m.RedditListTag
	}
	doc, err := goquery.NewDocument(url)
	if err != nil {
		log.Println("Failed to get reddit list subreddits for ", m.RedditListTag)
		return
	}
	var subs []string
	doc.Find(".result-item-slug a").Each(func(i int, s *goquery.Selection) {
		sub := strings.TrimPrefix(s.Text(), "/r/")
		subs = append(subs, sub)
	})
	m.Subreddits = append(m.Subreddits, subs...)
}

func (m *RedditSearch) preload(lurk reddit.Lurker, b *bot.Bot) {
	for {
		select {
		case <-m.close:
			return
		case <-time.After(2 * time.Second):
			m.mu.Lock()
			full := len(m.posts) == cap(m.posts)
			m.mu.Unlock()
			if full {
				continue
			}
			sub := m.Subreddits[rand.Intn(len(m.Subreddits))]
			for {
				post, err := lurk.Thread("/r/" + sub + "/random")
				if err != nil {
					log.Printf("Error while getting random post from %s: %v\n", sub, err)
					sub = m.Subreddits[rand.Intn(len(m.Subreddits))]
					continue
				}
				if m.Check != nil && !m.Check(post, b) {
					continue
				}
				m.mu.Lock()
				m.posts = append(m.posts, post)
				m.mu.Unlock()
				break
			}
		}
	}
}
