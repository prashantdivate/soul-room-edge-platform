import type { SimpleIcon } from "simple-icons";
import {
  siAnydesk, siApache, siCurl, siDocker, siFlatpak, siGit, siGnubash,
  siLinux, siMariadb, siMongodb, siMysql, siNginx, siNodedotjs, siOpenssl,
  siPostgresql, siPython, siRabbitmq, siRedis, siTrivy, siUbuntu,
} from "simple-icons";

const brands: { test: RegExp; icon: SimpleIcon }[] = [
  { test: /^nginx/i, icon: siNginx }, { test: /^(apache2|httpd)/i, icon: siApache },
  { test: /^mariadb/i, icon: siMariadb }, { test: /^mysql/i, icon: siMysql }, { test: /^(postgres|libpq)/i, icon: siPostgresql },
  { test: /^mongo/i, icon: siMongodb }, { test: /^redis/i, icon: siRedis }, { test: /^rabbitmq/i, icon: siRabbitmq },
  { test: /^(docker|containerd)/i, icon: siDocker }, { test: /^python/i, icon: siPython }, { test: /^git($|-)/i, icon: siGit },
  { test: /^ubuntu/i, icon: siUbuntu }, { test: /^flatpak/i, icon: siFlatpak }, { test: /^(openssl|libssl)/i, icon: siOpenssl },
  { test: /^(curl|libcurl)/i, icon: siCurl }, { test: /^(nodejs|node-|npm$)/i, icon: siNodedotjs }, { test: /^anydesk/i, icon: siAnydesk },
  { test: /^(bash|dash$)/i, icon: siGnubash }, { test: /^(linux|kernel)/i, icon: siLinux }, { test: /^trivy/i, icon: siTrivy },
];

export default function PackageBrandIcon({ name }: { name: string }) {
  const brand = brands.find((entry) => entry.test.test(name))?.icon;
  if (brand) return <span className="packageBrand" title={brand.title} style={{ color: `#${brand.hex}` }}><svg viewBox="0 0 24 24" role="img" aria-label={`${brand.title} logo`}><path fill="currentColor" d={brand.path} /></svg></span>;
  const initials = name.split(/[-_.]+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join("").toUpperCase() || "PK";
  const tone = Array.from(name).reduce((sum, letter) => sum + letter.charCodeAt(0), 0) % 5;
  return <span className={`packageMonogram tone${tone}`} title="System package without an official project logo">{initials}</span>;
}
