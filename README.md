# GeoIP For CN

市面上的 GeoIP 数据库基本都是同一个源的，IP 数据来自：

- MaxMind 的 [GeoLite2](https://dev.maxmind.com/geoip/geoip2/geolite2/)
- 基于 ipip.net 的 [17mon/china_ip_list](https://github.com/17mon/china_ip_list) 和 [gaoyifan/china-operator-ip](https://github.com/gaoyifan/china-operator-ip)
- 基于 纯真 IP 的 [metowolf/iplist](https://github.com/metowolf/iplist)
- [misakaio/chnroutes2](https://github.com/misakaio/chnroutes2)
- [ipverse/country-ip-blocks](https://github.com/ipverse/country-ip-blocks)
- [mayaxcn/china-ip-list](https://github.com/mayaxcn/china-ip-list)

MaxMind 的是最全的，不仅包含 CN，还有俄罗斯等各个国家。但是对于中国大陆的用户来说，最重要的是判断 IP 是否属于海外的（仅包含中国大陆，不包括香港、澳门、台湾），不需要乱七八糟的数据，但需要紧凑精简的大陆 IP 数据库。

本仓库的目的就是尽可能精确地、及时地收集更新中国大陆的 IPv4 和 IPv6。

## 下载

所有文件均发布到最新 GitHub Release，并同步到 `release` 分支供 jsDelivr CDN 使用。MMDB 用户建议下载双栈 `Country.mmdb`，sing-box 用户建议下载双栈 `cn.srs`。

### MMDB 数据库

| 版本                | 文件                | GitHub Release                                                                                | jsDelivr CDN                                                                              | 适用场景                                              |
| ------------------- | ------------------- | --------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ----------------------------------------------------- |
| IPv4 + IPv6（推荐） | `Country.mmdb`      | [GitHub 下载](https://github.com/JimmyHuang454/Geoip-For-CN/releases/latest/download/Country.mmdb)      | [jsDelivr 下载](https://cdn.jsdelivr.net/gh/JimmyHuang454/Geoip-For-CN@release/Country.mmdb)      | Clash、Surge、Shadowrocket、Quantumult X 等双栈客户端 |
| 仅 IPv4             | `Country-IPv4.mmdb` | [GitHub 下载](https://github.com/JimmyHuang454/Geoip-For-CN/releases/latest/download/Country-IPv4.mmdb) | [jsDelivr 下载](https://cdn.jsdelivr.net/gh/JimmyHuang454/Geoip-For-CN@release/Country-IPv4.mmdb) | 只需要 IPv4 判定的环境                                |
| 仅 IPv6             | `Country-IPv6.mmdb` | [GitHub 下载](https://github.com/JimmyHuang454/Geoip-For-CN/releases/latest/download/Country-IPv6.mmdb) | [jsDelivr 下载](https://cdn.jsdelivr.net/gh/JimmyHuang454/Geoip-For-CN@release/Country-IPv6.mmdb) | 只需要 IPv6 判定的环境                                |

### sing-box SRS 规则集

| 版本                | 文件          | GitHub Release                                                                          | jsDelivr CDN                                                                        |
| ------------------- | ------------- | --------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| IPv4 + IPv6（推荐） | `cn.srs`      | [GitHub 下载](https://github.com/JimmyHuang454/Geoip-For-CN/releases/latest/download/cn.srs)      | [jsDelivr 下载](https://cdn.jsdelivr.net/gh/JimmyHuang454/Geoip-For-CN@release/cn.srs)      |
| 仅 IPv4             | `cn-ipv4.srs` | [GitHub 下载](https://github.com/JimmyHuang454/Geoip-For-CN/releases/latest/download/cn-ipv4.srs) | [jsDelivr 下载](https://cdn.jsdelivr.net/gh/JimmyHuang454/Geoip-For-CN@release/cn-ipv4.srs) |
| 仅 IPv6             | `cn-ipv6.srs` | [GitHub 下载](https://github.com/JimmyHuang454/Geoip-For-CN/releases/latest/download/cn-ipv6.srs) | [jsDelivr 下载](https://cdn.jsdelivr.net/gh/JimmyHuang454/Geoip-For-CN@release/cn-ipv6.srs) |

### CIDR 文本列表

| 版本        | 文件                        | GitHub Release                                                                                        | jsDelivr CDN                                                                                      | 适用场景                    |
| ----------- | --------------------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | --------------------------- |
| 仅 IPv4     | `CN-ip-cidr.txt`            | [GitHub 下载](https://github.com/JimmyHuang454/Geoip-For-CN/releases/latest/download/CN-ip-cidr.txt)            | [jsDelivr 下载](https://cdn.jsdelivr.net/gh/JimmyHuang454/Geoip-For-CN@release/CN-ip-cidr.txt)            | IPv4 防火墙、路由表和规则集 |
| 仅 IPv6     | `CN-ipv6-cidr.txt`          | [GitHub 下载](https://github.com/JimmyHuang454/Geoip-For-CN/releases/latest/download/CN-ipv6-cidr.txt)          | [jsDelivr 下载](https://cdn.jsdelivr.net/gh/JimmyHuang454/Geoip-For-CN@release/CN-ipv6-cidr.txt)          | IPv6 防火墙、路由表和规则集 |
| IPv4 + IPv6 | `CN-ipv4-and-ipv6-cidr.txt` | [GitHub 下载](https://github.com/JimmyHuang454/Geoip-For-CN/releases/latest/download/CN-ipv4-and-ipv6-cidr.txt) | [jsDelivr 下载](https://cdn.jsdelivr.net/gh/JimmyHuang454/Geoip-For-CN@release/CN-ipv4-and-ipv6-cidr.txt) | 支持混合 CIDR 的双栈工具    |

> GitHub 下载链接始终指向最新 Release；jsDelivr CDN 在部分网络环境下下载更快，但可能存在短时间的缓存延迟。

## 使用方法

支持 MMDB 格式的工具都可以使用，对于 中国大陆的 IP 即返回 CN，非中国大陆的 IP 即返回 None。

### Clash/Clash Meta/Mihomo

需要先下载你想要使用的 .mmdb 格式文件，命名为 Country.mmdb，并放置在程序目录内。

```
rules:
  - GEOIP, CN, DIRECT
```

### sing-box

使用远程双栈 SRS 规则集：

```json
{
  "route": {
    "rules": [
      {
        "rule_set": "geoip-cn",
        "action": "route",
        "outbound": "direct"
      }
    ],
    "rule_set": [
      {
        "type": "remote",
        "tag": "geoip-cn",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/JimmyHuang454/Geoip-For-CN@release/cn.srs",
        "update_interval": "3d"
      }
    ]
  }
}
```

SRS 使用兼容 sing-box 1.8.0 及以上版本的二进制规则集格式。

### 从 CIDR 列表生成文件

输出格式默认根据目标文件扩展名识别：

```sh
go run . -s CN-ipv4-and-ipv6-cidr.txt -d Country.mmdb
go run . -s CN-ipv4-and-ipv6-cidr.txt -d cn.srs
```

也可以通过 `-format mmdb` 或 `-format srs` 显式指定格式。
