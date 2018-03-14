Undernet IRCbot
===============

IRCbot is a simple Undernet IRC bot written in [go](https://en.wikipedia.org/wiki/Golang).

Building and running
====================

First, clone the repo, cd into the newly created directory and pull all the `go` dependencies:

	$ git clone git@github.com:sizeofcat/ircbot.git && cd ircbot
	$ go get -d .

Secondly, build the binary:

	$ go build

Thirdly, rename `config/config-dist.json` to `config/config.json` and adjust the settings as needed.

As the final step, run the binary:

	$ ./ircbot

Configuration
=============

- Nickname - Nickname the bot will use.
- Username - X username to authenticate with. If left empty, the bot will not attempt to authenticate with X.
- Password - X password to authenticate with. If left empty, the bot will not attempt to authenticate with X.
- Channels - Autojoin the channels in the list.
- SSL - Use SSL to connect to the specified IRC server. Can be `true` or `false`.
- ServerHost - Undernet server to connect to.
- ServerPort - Server port to connect to (6667 or 6697 for SSL).
- Prefix - Command prefix (.help for example).
- Users - List of hostnames that have special access. Permissions can be `all`, `opcmds`, `invite` or `annoy`.
- ImageHosts - List of image hosting sites that will parse images from.

License
=======

Undernet IRCbot is written by sizeof(cat) <sizeofcat AT riseup DOT net> based on the [nag](https://github.com/noonien/nag) Snoonet IRC bot by [noonien](https://github.com/noonien/) and distributed under the [MIT license](LICENSE).
