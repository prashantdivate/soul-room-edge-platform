import type { SimpleIcon } from "simple-icons";
import { Package } from "lucide-react";
import {
  siAnsible, siAnydesk, siApache, siCaddy, siCanonical, siCmake, siContainerd,
  siCurl, siDebian, siDocker, siEclipsemosquitto, siFfmpeg, siFirefoxbrowser,
  siFlatpak, siFreedesktopdotorg, siGit, siGnome, siGnu, siGnubash, siGrafana, siInfluxdb, siJenkins, siK3s,
  siKubernetes, siLinux, siMariadb, siMongodb, siMysql, siNginx, siNodedotjs,
  siNpm, siOpenjdk, siOpenssl, siPerl, siPhp, siPodman, siPostgresql,
  siPrometheus, siPython, siQt, siRabbitmq, siRaspberrypi, siRedis, siRuby,
  siRust, siSnapcraft, siSqlite, siTailscale, siTerraform, siTrivy, siUbuntu,
  siVim, siVlcmediaplayer, siWayland, siWireguard, siXdotorg,
} from "simple-icons";

const brands: { test: RegExp; icon: SimpleIcon }[] = [
  { test: /^nginx/i, icon: siNginx }, { test: /^(apache2|httpd)/i, icon: siApache },
  { test: /^mariadb/i, icon: siMariadb }, { test: /^mysql/i, icon: siMysql }, { test: /^(postgres|libpq)/i, icon: siPostgresql },
  { test: /^mongo/i, icon: siMongodb }, { test: /^redis/i, icon: siRedis }, { test: /^rabbitmq/i, icon: siRabbitmq },
  { test: /^docker/i, icon: siDocker }, { test: /^containerd/i, icon: siContainerd }, { test: /^podman/i, icon: siPodman },
  { test: /^python/i, icon: siPython }, { test: /^git($|-)/i, icon: siGit }, { test: /^rust/i, icon: siRust },
  { test: /^ruby/i, icon: siRuby }, { test: /^perl/i, icon: siPerl }, { test: /^php/i, icon: siPhp },
  { test: /^(openjdk|java-)/i, icon: siOpenjdk }, { test: /^(nodejs|node-|libnode)/i, icon: siNodedotjs }, { test: /^npm($|-)/i, icon: siNpm },
  { test: /^ubuntu/i, icon: siUbuntu }, { test: /^(debian|dpkg)/i, icon: siDebian }, { test: /^canonical/i, icon: siCanonical },
  { test: /^flatpak/i, icon: siFlatpak }, { test: /^snapd/i, icon: siSnapcraft }, { test: /^(openssl|libssl)/i, icon: siOpenssl },
  { test: /^(curl|libcurl)/i, icon: siCurl }, { test: /^anydesk/i, icon: siAnydesk }, { test: /^(bash|dash$)/i, icon: siGnubash },
  { test: /^(linux|kernel)/i, icon: siLinux }, { test: /^trivy/i, icon: siTrivy }, { test: /^cmake/i, icon: siCmake },
  { test: /^sqlite/i, icon: siSqlite }, { test: /^qt/i, icon: siQt }, { test: /^vim/i, icon: siVim },
  { test: /^firefox/i, icon: siFirefoxbrowser }, { test: /^ffmpeg/i, icon: siFfmpeg }, { test: /^vlc/i, icon: siVlcmediaplayer },
  { test: /^mosquitto/i, icon: siEclipsemosquitto }, { test: /^grafana/i, icon: siGrafana }, { test: /^prometheus/i, icon: siPrometheus },
  { test: /^influxdb/i, icon: siInfluxdb }, { test: /^jenkins/i, icon: siJenkins }, { test: /^kube/i, icon: siKubernetes }, { test: /^k3s/i, icon: siK3s },
  { test: /^ansible/i, icon: siAnsible }, { test: /^terraform/i, icon: siTerraform }, { test: /^caddy/i, icon: siCaddy },
  { test: /^wireguard/i, icon: siWireguard }, { test: /^tailscale/i, icon: siTailscale }, { test: /^wayland/i, icon: siWayland },
  { test: /^xserver|^xorg/i, icon: siXdotorg }, { test: /^raspberrypi/i, icon: siRaspberrypi },
  { test: /^(accountsservice|appstream|avahi)/i, icon: siFreedesktopdotorg },
  { test: /^(adwaita|at-spi2|baobab|dconf|gdm|gnome|gtk|libgtk)/i, icon: siGnome },
  { test: /^(adduser|apt($|-)|base-files|base-passwd|debconf|dpkg)/i, icon: siDebian },
  { test: /^(apport|cloud-init|ubuntu)/i, icon: siUbuntu },
  { test: /^(aspell|autoconf|automake|binutils|coreutils|findutils|gawk|gcc|gettext|grep|gzip|libc6|make$|sed|tar)/i, icon: siGnu },
];

export function packageVisualKind(name: string) {
  if (brands.some((entry) => entry.test.test(name))) return "brand";
  return "system";
}

export default function PackageBrandIcon({ name }: { name: string }) {
  const brand = brands.find((entry) => entry.test.test(name))?.icon;
  if (brand) return <span className="packageBrand" title={brand.title}><svg viewBox="0 0 24 24" role="img" aria-label={`${brand.title} logo`} style={{ color: `#${brand.hex}` }}><path fill="currentColor" d={brand.path} /></svg></span>;
  return <span className="packageNoLogo" title={`${name} is a system package without published project artwork`} aria-label="System package"><Package size={23} strokeWidth={1.8} aria-hidden="true" /></span>;
}
