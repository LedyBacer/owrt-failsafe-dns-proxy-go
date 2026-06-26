# SPDX-License-Identifier: Apache-2.0

include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-failsafe-dns-proxy
PKG_VERSION:=0.1.0
PKG_RELEASE:=4

LUCI_TITLE:=LuCI support for Failsafe DNS Proxy
LUCI_DESCRIPTION:=Configure and monitor strict-priority DNS failover
LUCI_DEPENDS:=+luci-base +failsafe-dns-proxy
LUCI_PKGARCH:=all

PKG_LICENSE:=Apache-2.0
PKG_MAINTAINER:=Nikita Dybov

include $(TOPDIR)/feeds/luci/luci.mk

# call BuildPackage - OpenWrt buildroot signature
