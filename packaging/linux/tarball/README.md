# Pangolin client for Linux

This archive is for distributions without a Pangolin package (Gentoo, Void,
Alpine, Slackware, ...). On Debian, Ubuntu, Fedora, openSUSE or Arch, install
the deb, rpm or Arch package instead: those update themselves from the app.

## Requirements

- x86_64 or ARM64, glibc 2.35 or newer (musl systems need a glibc
  compatibility layer)
- GTK 3 and WebKitGTK 4.1 (Gentoo: `net-libs/webkit-gtk:4.1`; Void:
  `libwebkit2gtk41`)
- systemd or OpenRC to run the background service, or start
  `pangolin-client --daemon` as root yourself

## Install

    tar xzf pangolin-client-<arch>-<version>.tar.gz
    cd pangolin-client-<arch>-<version>
    sudo ./install.sh

This installs to `/usr/local` (set `PREFIX` to change it) and starts the
background service. Open Pangolin from your applications menu. On GNOME, log
out and back in, then run `gnome-extensions enable pangolin@pangolin.net` to
show Pangolin in the top bar.

Run `./install.sh --help` for the options, including `DESTDIR` for building a
distribution package from this archive.

## Update

The app tells you when a new version is out but can't install it from this
archive. Download the new archive and run its `install.sh` again: it replaces
the files and restarts the service, keeping your sign-ins.

## Uninstall

    sudo /usr/local/lib/pangolin-client/uninstall.sh

Your sign-ins and logs stay in `/var/lib/pangolin-client` and
`/var/log/pangolin-client`; delete those too to remove them.

## What's inside

| File | Installed to |
| --- | --- |
| `bin/pangolin-client` | `$PREFIX/bin/` |
| `share/applications/pangolin-client.desktop` | `$PREFIX/share/applications/` |
| `share/icons/hicolor/256x256/apps/pangolin-client.png` | `$PREFIX/share/icons/hicolor/256x256/apps/` |
| `share/gnome-shell/extensions/pangolin@pangolin.net/` | `$PREFIX/share/gnome-shell/extensions/` |
| `share/polkit-1/actions/net.pangolin.client.policy` | `/usr/share/polkit-1/actions/` |
| `etc/xdg/autostart/pangolin-client-autostart.desktop` | `/etc/xdg/autostart/` |
| `lib/systemd/system/pangolin-client.service` | `/etc/systemd/system/` (systemd) |
| `etc/init.d/pangolin-client` | `/etc/init.d/` (OpenRC) |
