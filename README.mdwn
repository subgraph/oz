# Intro

Oz is a sandboxing system targeting everyday workstation applications.
It acts as a wrapper around application executables for completely transparent user operations. It achieves process containment through the use of [Linux Namespaces](http://man7.org/linux/man-pages/man7/namespaces.7.html), [Seccomp filters](http://man7.org/linux/man-pages/man2/seccomp.2.html), [Capabilities](http://man7.org/linux/man-pages/man7/capabilities.7.html), and X11 restriction using [Xpra](https://xpra.org/). It has built-in support with automatic configuration of bridge mode networking and also support working with contained network environment using the built in connection forwarding proxy.

[See the wiki for the complete technical documentation.](https://github.com/subgraph/oz/wiki/Oz-Technical-Details)

# Demo

<p align="center">
<a href="https://support.subgraph.com/videos/oz_evince_01.webm"><img src="https://raw.githubusercontent.com/subgraph/oz/docs/videos/oz_evince_01.gif" alt="OZ Sandbox Evince Demo"/></a>
</p>

# Warnings!

Please note that Oz is currently under major development and still at a very alpha stage. **As of this writing some features are not yet available in the public master branch**. It is not intended for use in multi-users systems. Use it at your own risk!

# Installing

You can easily build a debian package from the latest tagged release:

	$ sudo apt-get install golang xpra bridge-utils ebtables libacl1 libacl1-dev dh-golang dh-systemd git-buildpackage
	$ git clone -b debian https://github.com/subgraph/oz.git
	$ cd oz
	### To Build from stable
	$ gbp buildpackage -us -uc
	### To Build from strict-etc
	$ gbp buildpackage -us -uc --git-upstream-branch=strict-etc
	### Once you are ready to install and have no important unsaved content in Oz windows
	$ sudo dpkg -i /tmp/build-area/oz-daemon_<version>.deb
	$ sudo paxrat
	$ sudo systemctl restart oz-daemon.service

Or to build it manually:

## Prerequisites

Currently Oz only works and is only tested for Debian (>= jessie).
As of this writing it does **not** work in Ubuntu. While it has been minimally tested under Fedora,
there doesn't exist an RPM equivalent to `dpkg-divert` which allows to conveniantly wrap executables of existing program.
For seccomp filters you need at least kernel version 3.17. We recommend using Debian stretch with a 4.0+ grsecurity patched kernel.

It is highly recommend that you run it in conjunction with [grsecurity](https://grsecurity.net/).

### Dependencies

```
$ sudo apt-get install golang xpra bridge-utils ebtables libacl1
```

You need golang version 1.4.

You must also have the `veth` and `bridge` kernel module loaded to use *bridge network mode*.

### Grsec

If you are using grsecurity you will need to disable the following kernel options
permanently via the sysctl interface or by echoing values to files in `/proc/sys/kernel/grsecurity/`:

```
kernel.grsecurity.chroot_caps = 0
kernel.grsecurity.chroot_deny_chmod = 0
kernel.grsecurity.chroot_deny_mount = 0
```

See [Grsecurity/Appendix/Sysctl Options](https://en.wikibooks.org/wiki/Grsecurity/Appendix/Sysctl_Options)
for information on setting grsecurity sysctl options.

### Network Manager

If you are using Network-Manager you need to make sure to exclude the bridge interface.
To do so a file must be created in `/etc/NetworkManager/conf.d/oz.conf` containing:

```
[main]
plugins=keyfile

[keyfile]
unmanaged-devices=mac:6a:a8:2e:56:e8:9c;interface-name:oz0
```

If this file is missing `oz-daemon` will output a warning but it may fail to setup bridge networking.

### Bridge networking

Bridge networking is automatically configured by Oz, but you will need to setup
at minimum a few iptables masquerading rules and ebtables isolation rules as follows (adjusted with the name of the interfaces):

```
sudo iptables -t nat -A POSTROUTING -o $INTERFACE_ONE -j MASQUERADE
sudo iptables -t nat -A POSTROUTING -o $INTERFACE_TWO -j MASQUERADE
sudo ebtables -P FORWARD DROP
sudo ebtables -F FORWARD
sudo ebtables -A FORWARD -i oz0 -j ACCEPT
sudo ebtables -A FORWARD -o oz0 -j ACCEPT
```

IP Forwarding must be enabled (do this as root):

```
# echo 1 >/proc/sys/net/ipv4/ip_forward
```

## Building

1. To setup a GOPATH for Oz, run the following commands (or you can use your
existing GOPATH):

```
$ export GOPATH=/opt/local/golang-oz
$ sudo mkdir -p $GOPATH
$ sudo chown user. $GOPATH
```

2. To build Oz with the required external Golang dependencies, run the following
commands:

Firstly we build the [godep](https://github.com/tools/godep) tool:

```
$ go get github.com/tools/godep
```

Now checkout the source using either:

```
$ git clone https://github.com/subgraph/oz.git src/github.com/subgraph/oz
```

Or alternatively if we want to create a development environment use:

```
$ go get -d github.com/subgraph/oz
```

We are ready to build and install:

``` 
$ sudo apt-get install libacl1-dev
$ cd $GOPATH/src/github.com/subgraph/oz/
$ $GOPATH/bin/godep go install ./...
$ sudo cp $GOPATH/bin/oz* /usr/local/bin
$ sudo mkdir -p /var/lib/oz/cells.d
$ sudo cp $GOPATH/src/github.com/subgraph/oz/profiles/* /var/lib/oz/cells.d/
$ sudo mkdir /etc/oz
$ sudo cp $GOPATH/src/github.com/subgraph/oz/profiles/generic-blacklist.seccomp /etc/oz/blacklist-generic.seccomp
$ sudo cp $GOPATH/src/github.com/subgraph/oz/sources/etc/network/if-up.d/* /etc/network/if-up.d/
$ sudo chmod a+x /etc/network/if-up.d/oz
$ sudo cp $GOPATH/src/github.com/subgraph/oz/sources/etc/network/if-post-down.d/* /etc/network/if-post-down.d/
$ sudo cp $GOPATH/src/github.com/subgraph/oz/sources/lib/systemd/system/oz-daemon.service /lib/systemd/system/oz-daemon.service
$ sudo cp $GOPATH/src/github.com/subgraph/oz/sources/etc/NetworkManager/conf.d/oz.conf /etc/NetworkManager/conf.d/oz.conf
$ sudo dpkg-divert --add --package oz --rename --divert /etc/xpra/xpra.conf.pkg-dist /etc/xpra/xpra.conf
$ sudo cp $GOPATH/src/github.com/subgraph/oz/sources/etc/xpra/xpra.conf /etc/xpra/xpra.conf
$ sudo systemctl enable oz-daemon.service
```

## Enabling a profile

Once installed you can enable profiles using the `oz-setup` command as such (using the profile for the 'evince' application):

```
$ sudo oz-setup install evince
```

This will install a symlink to the oz client executable in place of the original, the original executable will be moved to `dirname(path)-oz/basename(path).unsafe`. This behavior can be altered using the divert_path and divert_suffix configuration flags.

## Disabling a profile

If you want to disable a profile that you're not using or prior to uninstalling Oz, you can run the following command (once again
with 'evince' as the target application):


```
$ sudo oz-setup remove evince
```

If you have already uninstalled Oz you can remove the divertions manually using the `dpkg-divert` command.

## Viewing the status of a profile

You can view the status of a profile using the `status` sub-command:

```
$ sudo oz-setup status /usr/bin/lowriter
Package divert is installed for:     /usr/bin/libreoffice
Package divert is installed for:     /usr/bin/lowriter
Package divert is installed for:     /usr/bin/lobase
Package divert is installed for:     /usr/bin/localc
Package divert is installed for:     /usr/bin/loffice
<...output truncated for clarity>
```

# Usage

Firstly you must launch the daemon utility either with the init script or manually:

```
$ sudo oz-daemon
```

Once the daemon is started you can transparently launch any applications for which you have enabled the profile.
This means Oz sandboxing will be used whether you launch your browser from gnome-shell or from the command line.

Any files inside of your home passed as arguments to the command (either via double clicking or program arguments) are automatically added to the whitelist (if the profile supports `allow_files`).

The [OZ gnome-shell extension](https://github.com/subgraph/ozshell-gnome-extension) allows you to easily interface running sandboxes:
to add/remove files inside a sandbox, open a shell inside a sandbox, and terminate a sandbox.

If you wish to run an executable outside of the sandbox simply call it at `dirname(path)-oz/basename(path).unsafe`, for example:

```
$ /usr/bin-oz/evince.unsafe
```

# Advanced Usage Information

## Oz client commands

The `oz` executable acts as a client for the daemon when called directly. It provides a number of commands to interact with sandboxes.

* `profiles`: lists available profiles
* `launch <name>`: launches a sandbox for the given profile name, pass the `--noexec` flag to prevent execution of the default program
* `list`: lists the running sandboxes
* `kill <id>`: kills the sandbox with the given numerical id
* `kill all`: kills all running sandboxes
* `shell <id>`: enters a shell in a given sandbox, mostly useful for debugging
* `logs [-f]`: prints out the logs, pass `-f` to follow the output

## Oz-daemon configurations

In nearly every case the default configurations should be used, but for debugging and development purposes some flags are configurable inside of the `/etc/oz/oz.conf` file. You can view the current configuration by running the following command:

```
$ oz-setup config show

Config file     : /etc/oz/oz.conf
##################################################################
profile_dir     : /var/lib/oz/cells.d                            # Directory containing the sandbox profiles
shell_path      : /bin/bash                                      # Path of the shell used when entering a sandbox
prefix_path     : /usr/local                                     # Prefix path containing the oz executables
etc_prefix      : /etc/oz                                        # Prefix for configuration files
sandbox_path    : /srv/oz                                        # Path of the sandboxes base
bridge_mac      : 6A:A8:2E:56:E8:9C                              # MAC Address of the bridge interface
divert_suffix   : unsafe                                         # Suffix using for dpkg-divert of application executables, can be left empty when using a divert path
divert_path     : true                                           # Whether the diverted executable should be moved out of the path
nm_ignore_file  : /etc/NetworkManager/conf.d/oz.conf             # Path to the NetworkManager ignore config file, disables the warning if empty
use_full_dev    : false                                          # Give sandboxes full access to devices instead of a restricted set
allow_root_shell: false                                          # Allow entering a sandbox shell as root
log_xpra        : false                                          # Log output of Xpra
environment_vars: [USER USERNAME LOGNAME LANG LANGUAGE _ TZ=UTC] # Default environment variables passed to sandboxes
default_groups  : [audio video]                                  # List of default group names that can be used inside the sandbox
```

## Profiles

Profiles files are simple JSON files located, by default, in `/var/lib/oz/cells.d`. They must include at minimum the path to the executable to be sandboxed using the `path` key. It may also define more executables to run under the same sandbox under the `paths` array; in which case a `name` key must also be specified. Some other base options are also available:

* `path`: if multiple executables are to be sandboxed under the same profile
* `allow_files`: whether to allow binding of files passed as arguments inside the sandbox (does not affect files added manually)
* `auto_shutdown`: whether the sandbox should be terminated right away after the process exits, one of [yes|no], (defaults to `yes`)
* `watchdog`: an array of strings containing the names of process the auto-shutdown feature should look for in case the main process spawns a detached process.
* `allowed_groups`: an array of user groups assigned to the user inside the sandbox
* `default_params`: an array of default params to pass to the program whenever it is executed

### Xserver

This section defines the configuration of the Xserver (namely [xpra](https://www.xpra.org/)).
Possible options are:

* `enabled`: whether or not to use the Xserver
* `enable_tray`: whether or not to enable the Xpra tray diagnostic menu/tray (This requires the [`Top Icons`](https://extensions.gnome.org/extension/495/topicons/) gnome-shell extension!)
* `tray_icon`: the path to an icon file to use for the to tray menu
* `window_icon`: the path to an icon file to use for windows
* `audio_mode`: one of [none|pulseaudio~~|speaker|full~~] selects the audio passthrough mode (defaults: none) (Only pulseaudio mode supported at this time)
* `disable_clipboard`: optionally disable clipboard sharing
* `enable_notifications`: enable passing of dbus notifications

### Network configs

The network can be configured in one of three different ways: host, bridge, and empty namespace, as defined in the `type` key.

* `empty`: the sandbox will live with an empty network namespace (ie: only `lo` interface)
* `bridge`: the sandbox will have its own network namespace and use *veth* to join a bridge named `oz0`
* `none`: don't even configure the loopback interface, connection proxy will be unavailable
* `host`: the sandbox will share the network namespace with the host (usually not desirable)


#### Port Forwarding config

Oz allows you to forward ports on the loopback interface between the host and the sandbox.
This is useful so that you can expose some services (such as a socks proxy) to the sandbox without giving the sandbox any real network access.
You may define as many forwards, called `sockets` in the configuration, as you like.
Each socket configuration contains the following keys:

* `type`: One of `client`, or `server`, this defines whether to connect (*client*) or listen (*server*) on the host side
* `proto`: One of `tcp`, or `udp`, or `socket`
* `port`: The network port number to connect to
* `destination`: *Optional*, in client mode this is the address to connect to, in server mode this is the address to bind to. Defaults to *localhost*.


### Bind list

There exists two types of *bindlists*: a whitelist and a blacklist.
The whitelist allows you to bring files from the host into the sandbox; while the blacklist allows you to remove access to specific files or directories.

The `path` key specifies the path of the directory or file to bind. 

Both of these types support a few ways of resolving files:

* In the path by using the `${PATH}` prefix.
* In the home by using the `${HOME}` prefix.
* By replacing `${UID}` with the user numeric id.
* By replacing `${USER}` with the user login.
* By replacing any `${XDG_<DIRECTORY>_DIR}` with the current localized version of that XDG directory.
* By path globbing using the `*` wildcard.


The whitelist carries some extra properties:

* An optional `target` key can be specified to bind the file to a different path inside the sandbox.
* If the original file does not exist and is inside the home, an empty directory will be created in its place if the `can_create` key is set.
* If the target already exists the whitelist will fail to bind unless the `force` key is set.
* A profile will fail to launch if a whitelist item is missing unless the `ignore` key is set.
* An item can be marked as read only with the `read_only` boolean key.
* Files passed as arguments to the command while launching are automatically added to the whitelist (if the `allow_files` boolean key is set).

The whitelist carries some extra caveats:

* If the original file is a symlink it is resolved, but the target remains the same.

### Environment

One can specify which environment variables to pass by defining them in this list.
It is also possible to define static variables by also defining a `value` attribute in the list item.

### Seccomp 

Oz supports both whitelist and blacklist seccomp policies for sandboxed applications. Seccomp allows for See the [Oz Seccomp documentation page](https://github.com/subgraph/oz/wiki/Oz-Seccomp) for more details.

Oz can also run sandboxed applications with whitelist and blacklist seccomp policies loaded, but in non-enforced (audit only) mode. More information is available on the [Oz seccomp non-enforcement mode documentation](https://github.com/subgraph/oz/wiki/Oz-Seccomp-Non-Enforcement-Mode) page.

### Example

You can find a list of existing profiles in the repository. Here is the porfile for running the `torbrowser-launcher`:

```
{
"path": "/usr/bin/torbrowser-launcher"
, "watchdog": ["start-tor-browser", "firefox"]
, "xserver": {
	"enabled": true
	, "enable_tray": false
	, "tray_icon":"/usr/share/pixmaps/torbrowser80.xpm"
	, "audio_mode": "pulseaudio"
}
, "networking":{
	"type":"empty"
	, "sockets": [
		{"type":"client", "proto":"tcp", "port":9050}
		, {"type":"client", "proto":"tcp", "port":9051}
	]
}
, "whitelist": [
	{"path":"${HOME}/.local/share/torbrowser", "can_create":true}
	, {"path":"${HOME}/.config/torbrowser", "can_create":true}
	, {"path":"${HOME}/Downloads/TorBrowser", "can_create":true}
	, {"path":"/run/tor/control.authcookie", "ignore":true}
]
, "blacklist": [
]
, "environment": [
	{"name":"TOR_SKIP_LAUNCH"}
	, {"name":"TOR_SOCKS_HOST"}
	, {"name":"TOR_SOCKS_PORT"}
	, {"name":"TOR_CONTROL_PORT"}
	, {"name":"TOR_CONTROL_PASSWD"}
	, {"name":"TOR_CONTROL_AUTHENTICATE"}
	, {"name":"TOR_CONTROL_COOKIE_AUTH_FILE"}
]
, "allowed_groups": ["debian-tor"]
, "seccomp": {
	"mode":"blacklist"
	, "enforce": true
}
}
```

# FAQ

**Q: Why don't you use unprivileged user namespaces, I hear it would be far more secure!**

A: Please see [this issue which explains the reasoning and tracks multiple unprivileged user namespace related kernel vulnerabilities](https://github.com/subgraph/oz/issues/11#issuecomment-163396758)


<p align="center">
<img src="https://raw.githubusercontent.com/subgraph/oz/docs/images/oz_logo_02.png" alt="Obsidian Zebra" />
</p>
