# SPDX-License-Identifier: MIT

include $(TOPDIR)/rules.mk

PKG_NAME:=failsafe-dns-proxy
PKG_VERSION:=0.1.0
PKG_RELEASE:=15

PKG_LICENSE:=MIT
PKG_LICENSE_FILES:=LICENSE
PKG_MAINTAINER:=Nikita Dybov
PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_PARALLEL:=1
PKG_BUILD_FLAGS:=no-mips16
PKG_BUILD_DIR:=$(BUILD_DIR)/$(PKG_NAME)-$(PKG_VERSION)

GO_PKG:=github.com/nikitadybov/owrt-failsafe-dns-proxy-go
GO_PKG_BUILD_PKG:=$(GO_PKG)/cmd/failsafe-dns-proxy
GO_PKG_LDFLAGS:=-s -w
GO_PKG_LDFLAGS_X:=main.version=$(PKG_VERSION)-r$(PKG_RELEASE)

FDP_SOURCE_DIR?=$(abspath $(CURDIR)/../..)

include $(INCLUDE_DIR)/package.mk
include $(TOPDIR)/feeds/packages/lang/golang/golang-package.mk

GO_PKG_VARS+=CGO_ENABLED=0
GO_PKG_DEFAULT_LDFLAGS=-buildid '$(SOURCE_DATE_EPOCH)'

define Package/failsafe-dns-proxy
  SECTION:=net
  CATEGORY:=Network
  SUBMENU:=IP Addresses and Names
  TITLE:=Strict-priority DNS failover proxy
  URL:=https://github.com/nikitadybov/owrt-failsafe-dns-proxy-go
  DEPENDS:=$(GO_ARCH_DEPENDS)
endef

define Package/failsafe-dns-proxy/description
 A small UDP/TCP DNS forwarding daemon with persistent in-memory health state,
 strict upstream priority, bounded failover, active probes, and failback.
endef

define Package/failsafe-dns-proxy/conffiles
/etc/config/failsafe-dns-proxy
endef

define Build/Prepare
	$(call Build/Prepare/Default)
	$(CP) $(FDP_SOURCE_DIR)/cmd $(PKG_BUILD_DIR)/
	$(CP) $(FDP_SOURCE_DIR)/internal $(PKG_BUILD_DIR)/
	$(CP) $(FDP_SOURCE_DIR)/go.mod $(FDP_SOURCE_DIR)/go.sum $(PKG_BUILD_DIR)/
	$(CP) $(FDP_SOURCE_DIR)/LICENSE $(PKG_BUILD_DIR)/
endef

define Package/failsafe-dns-proxy/install
	$(INSTALL_DIR) $(1)/usr/sbin
	$(INSTALL_BIN) $(GO_PKG_BUILD_BIN_DIR)/failsafe-dns-proxy $(1)/usr/sbin/failsafe-dns-proxy
	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) ./files/etc/config/failsafe-dns-proxy $(1)/etc/config/failsafe-dns-proxy
	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/etc/init.d/failsafe-dns-proxy $(1)/etc/init.d/failsafe-dns-proxy
	$(INSTALL_DIR) $(1)/usr/sbin
	$(INSTALL_BIN) ./files/usr/sbin/failsafe-dns-proxy-dnsmasq $(1)/usr/sbin/failsafe-dns-proxy-dnsmasq
	$(INSTALL_BIN) ./files/usr/sbin/failsafe-dns-proxy-soak $(1)/usr/sbin/failsafe-dns-proxy-soak
endef

define Package/failsafe-dns-proxy/prerm
#!/bin/sh
[ "$${PKG_UPGRADE:-0}" = "1" ] && exit 0
[ -x "$${IPKG_INSTROOT}/usr/sbin/failsafe-dns-proxy-dnsmasq" ] || exit 0
"$${IPKG_INSTROOT}/usr/sbin/failsafe-dns-proxy-dnsmasq" disable >/dev/null 2>&1 || exit 1
exit 0
endef

$(eval $(call GoBinPackage,failsafe-dns-proxy))
$(eval $(call BuildPackage,failsafe-dns-proxy))
