SUMMARY = "Soul Room edge device management agent"
LICENSE = "MIT"
LIC_FILES_CHKSUM = "file://LICENSE;md5=0835ade698e0bcf8506ecda2f7b4f302"

SRC_URI = "git://example.invalid/soul-room/edge-agent.git;protocol=https;branch=main"
SRCREV = "${AUTOREV}"

S = "${WORKDIR}/git"

inherit go systemd

GO_IMPORT = "github.com/soul-room/edge-agent"
SYSTEMD_SERVICE:${PN} = "edge-agent.service"

do_install:append() {
    install -d ${D}${bindir}
    install -m 0755 ${B}/bin/edge-agent ${D}${bindir}/edge-agent
    install -m 0755 ${B}/bin/edge-agentctl ${D}${bindir}/edge-agentctl
    install -d ${D}${sysconfdir}/edge-agent
    install -m 0640 ${S}/configs/edge-agent.yaml ${D}${sysconfdir}/edge-agent/config.yaml
}
