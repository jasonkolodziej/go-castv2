# AirPlay 2 Protocol
- [AirPlay 2 Protocol](#airplay-2-protocol)
    - [Content from `avahi-utils`](#content-from-avahi-utils)
      - [Multi-room support](#multi-room-support)
    - [Features](#features)
    - [TXT Records](#txt-records)
      - [Airplay TCP](#airplay-tcp)
      - [RAOP TCP](#raop-tcp)
    - [Authentication/Encryption methods](#authenticationencryption-methods)
      - [Disabled authentication](#disabled-authentication)
      - [Encryption](#encryption)
      - [Disabled Encryption](#disabled-encryption)
    - [Audio Formats](#audio-formats)
      - [Audio Codecs](#audio-codecs)
    - [Metadata](#metadata)
      - [CanBeRemoteControlled](#canberemotecontrolled)
      - [State Changes](#state-changes)
    - [`GET /info`](#get-info)

> Reference: https://emanuelecozzi.net/docs/airplay2/rtsp/
>
> https://openairplay.github.io/airplay-spec/service_discovery.html#canberemotecontrolled

### Content from `avahi-utils`
```shell
avahi-browse -d local _airplay._tcp --resolve

=  wlan0 IPv6 Workshop-pi                                   AirPlay Remote Video local
   hostname = [workshop-pi.local]
   address = [fe80::da3a:ddff:fe9c:f879]
   port = [7000]
   txt = ["pk=caedeee3152d54079b33439e0d5830bbdcd661b892032e4bf6c89480f71b2e56" "gcgl=0" "gid=24fefbff-2927-4a2e-8d77-6742325de724" "pi=24fefbff-2927-4a2e-8d77-6742325de724" "model=Shairport Sync" "fv=4.3.2-2-g165431a8" "rsf=0x0" "acl=0" "protovers=1.1" "flags=0x4" "features=0x405FCA00,0x1C340" "deviceid=d8:3a:dd:9c:f8:79" "srcvers=366.0"]
=  wlan0 IPv4 k room                                        AirPlay Remote Video local
   hostname = [CastTV.local]
   address = [192.168.2.38]
   port = [7000]
   txt = ["pk=93d25574f62b3f78f00ab0e417cc0f2e160427e7455f37692637a631969e1112" "gcgl=0" "gid=00000000-0000-0000-0000-32B541FD4243" "psi=00000000-0000-0000-0000-32B541FD4243" "pi=32:B5:41:FD:42:43" "srcvers=377.40.00" "protovers=1.1" "serialNumber=LINID4KX2906323" "manufacturer=VIZIO Inc." "integrator=VIZIO Inc." "model=D40f-J09" "flags=0x244" "at=0x1" "fv=p20.3.600.31.1-5" "rsf=0x3" "fex=0Ip/AEbLCwBACA" "features=0x7F8AD0,0xBCB46" "deviceid=32:B5:41:FD:42:43" "acl=0"]
=  wlan0 IPv4 Workshop-pi                                   AirPlay Remote Video local
   hostname = [workshop-pi.local]
   address = [192.168.2.247]
   port = [7000]
   txt = ["pk=caedeee3152d54079b33439e0d5830bbdcd661b892032e4bf6c89480f71b2e56" "gcgl=0" "gid=24fefbff-2927-4a2e-8d77-6742325de724" "pi=24fefbff-2927-4a2e-8d77-6742325de724" "model=Shairport Sync" "fv=4.3.2-2-g165431a8" "rsf=0x0" "acl=0" "protovers=1.1" "flags=0x4" "features=0x405FCA00,0x1C340" "deviceid=d8:3a:dd:9c:f8:79" "srcvers=366.0"]

```
#### Multi-room support
The minimal set of features an AirPlay 2 receiver must declare for multi-room support are:

- `SupportsAirPlayAudio` (bit 9)
- `AudioRedundant` (bit 11)
- `HasUnifiedAdvertiserInfo` | `ROAP` (bit 30)
- `SupportsBufferedAudio` (bit 40)
- `SupportsPTP` (bit 41)
- `SupportsUnifiedPairSetupAndMFi` (bit 51)
  
>The respective features bitmask is `0x8030040000a00` and will be declared as `features=0x40000a00,0x80300` in a TXTRecord.

Example of **WORKING** Raspberry Pi Zero 2W:
> See [calc](https://openairplay.github.io/airplay-spec/features.html)

>>**Decimal**: 496155702053376   
**Hex**: 0x1C340405FCA00  
**mDNS**: 0x405FCA00,0x1C340

### Features
| ShairPort | bit | name                                  | description                                                                                                                         |
| --------- | --- | ------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
|           | 0   | Video                                 | video supported                                                                                                                     |
|           | 1   | Photo                                 | photo supported                                                                                                                     |
|           | 2   | VideoFairPlay                         | video protected with FairPlay DRM                                                                                                   |
|           | 3   | VideoVolumeControl                    | volume control supported for videos                                                                                                 |
|           | 4   | VideoHTTPLiveStreams                  | http live streaming supported                                                                                                       |
|           | 5   | Slideshow                             | slideshow supported                                                                                                                 |
|           | 6   |                                       |                                                                                                                                     |
|           | 7   | Screen                                | mirroring supported                                                                                                                 |
|           | 8   | ScreenRotate                          | screen rotation supported                                                                                                           |
| *         | 9   | Audio                                 | audio supported                                                                                                                     |
|           | 10  |                                       |                                                                                                                                     |
| *         | 11  | AudioRedundant                        | audio packet redundancy supported                                                                                                   |
|           | 12  | FPSAPv2pt5_AES_GCM                    | FairPlay secure auth supported                                                                                                      |
|           | 13  | PhotoCaching                          | photo preloading supported                                                                                                          |
| *         | 14  | Authentication4                       | Authentication type 4. FairPlay authentication                                                                                      |
| *         | 15  | MetadataFeature1                      | bit 1 of MetadataFeatures. Artwork.                                                                                                 |
| *         | 16  | MetadataFeature2                      | bit 2 of MetadataFeatures. Progress.                                                                                                |
| *         | 17  | MetadataFeature0                      | bit 0 of MetadataFeatures. Text.                                                                                                    |
| *         | 18  | AudioFormat1                          | support for audio format 1                                                                                                          |
| *         | 19  | AudioFormat2                          | support for audio format 2. This bit must be set for AirPlay 2 connection to work                                                   |
| *         | 20  | AudioFormat3                          | support for audio format 3. This bit must be set for AirPlay 2 connection to work                                                   |
|           | 21  | AudioFormat4                          | support for audio format 4                                                                                                          |
| *         | 22  |                                       |                                                                                                                                     |
|           | 23  | Authentication1                       | Authentication type 1. RSA Authentication                                                                                           |
|           | 24  |                                       |                                                                                                                                     |
|           | 25  |                                       |                                                                                                                                     |
|           | 26  | HasUnifiedAdvertiserInfo              |                                                                                                                                     |
|           | 27  | SupportsLegacyPairing                 |                                                                                                                                     |
|           | 28  |                                       |                                                                                                                                     |
|           | 29  |                                       |                                                                                                                                     |
| *         | 30  | RAOP                                  | RAOP is supported on this port. With this bit set your don't need the AirTunes service                                              |
|           | 31  |                                       |                                                                                                                                     |
|           | 32  | IsCarPlay / SupportsVolume            | Don’t read key from pk record it is known                                                                                           |
|           | 33  | SupportsAirPlayVideoPlayQueue         |                                                                                                                                     |
|           | 34  | SupportsAirPlayFromCloud              |                                                                                                                                     |
|           | 35  |                                       |                                                                                                                                     |
|           | 36  |                                       |                                                                                                                                     |
|           | 37  |                                       |                                                                                                                                     |
| *         | 38  | SupportsCoreUtilsPairingAndEncryption | SupportsHKPairingAndAccessControl, SupportsSystemPairing and SupportsTransientPairing implies SupportsCoreUtilsPairingAndEncryption |
|           | 39  |                                       |                                                                                                                                     |
| *         | 40  | SupportsBufferedAudio                 | Bit needed for device to show as supporting multi-room audio                                                                        |
| *         | 41  | SupportsPTP                           | Bit needed for device to show as supporting multi-room audio                                                                        |
|           | 42  | SupportsScreenMultiCodec              |                                                                                                                                     |
|           | 43  | SupportsSystemPairing                 |                                                                                                                                     |
|           | 44  |                                       |                                                                                                                                     |
|           | 45  |                                       |                                                                                                                                     |
| *         | 46  | SupportsHKPairingAndAccessControl     |                                                                                                                                     |
| *         | 47  |                                       |                                                                                                                                     |
| *         | 48  | SupportsTransientPairing              | SupportsSystemPairing implies SupportsTransientPairing                                                                              |
|           | 49  |                                       |                                                                                                                                     |
|           | 50  | MetadataFeature4                      | bit 4 of MetadataFeatures. binary plist.                                                                                            |
| **M**     | 51  | SupportsUnifiedPairSetupAndMFi        | Authentication type 8. MFi authentication                                                                                           |
|           | 52  | SupportsSetPeersExtendedMessage       |                                                                                                                                     |


### TXT Records
#### Airplay TCP
> The following fields are available in the `_airplay._tcp` TXT record

| name         | type                                                    | description                                                                                                                                                                                                                                                               |
| ------------ | ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| model        | string                                                  | device model                                                                                                                                                                                                                                                              |
| manufacturer | string                                                  | device manufacturer                                                                                                                                                                                                                                                       |
| serialNumber | string                                                  | device serial number                                                                                                                                                                                                                                                      |
| fv           | string                                                  | device firmware version                                                                                                                                                                                                                                                   |
| osvers       | string                                                  | device OS version                                                                                                                                                                                                                                                         |
| deviceid     | string                                                  | Device ID. Usually MAC address of the device                                                                                                                                                                                                                              |
| features     | 32 bit hex number,optional high order 32 bit hex number | bitfield of supported features. This was originally a 32 bit value but it has since been expanded to a 64 bit value. To support both these types the mDNS value is encoded as two 32 bit values separated by comma with the comma and second 32 bit value being optional. |
| pw           | boolean                                                 | server is password protected                                                                                                                                                                                                                                              |
| acl          | int64                                                   | Access control level                                                                                                                                                                                                                                                      |
| srcvers      | string                                                  | airplay version                                                                                                                                                                                                                                                           |
| flags        | 20 bit hex number                                       | bitfield of status flags                                                                                                                                                                                                                                                  |
| pk           | hex string                                              | public key                                                                                                                                                                                                                                                                |
| pi           | UUID string                                             | group_id / Public CU AirPlay Pairing Identifier                                                                                                                                                                                                                           |
| psi          | UUID string                                             | Public CU System Pairing Identifier                                                                                                                                                                                                                                       |
| gid          | UUID string                                             | group UUID                                                                                                                                                                                                                                                                |
| gcgl         | boolean                                                 | group contains group leader / Group contains discoverable leader                                                                                                                                                                                                          |
| igl          | boolean                                                 | is group leader                                                                                                                                                                                                                                                           |
| gpn          | string                                                  | group public name                                                                                                                                                                                                                                                         |
| hgid         | UUID string                                             | home group UUID                                                                                                                                                                                                                                                           |
| hmid         | string                                                  | household ID                                                                                                                                                                                                                                                              |
| pgcgl        | boolean                                                 | parent group contains discoverable leader                                                                                                                                                                                                                                 |
| pgid         | UUID string                                             | parent group UUID                                                                                                                                                                                                                                                         |
| tsid         | UUID string                                             | 3008B5C8-9BD3-4479-A564-90BFB3D780C0                                                                                                                                                                                                                                      |
| rsf          | 64 bit hex number                                       | required sender features                                                                                                                                                                                                                                                  |
| protovers    | string                                                  | protocol version                                                                                                                                                                                                                                                          |
| vv           | ?                                                       | vodka version                                                                                                                                                                                                                                                             |


#### RAOP TCP
> The following fields are available in the `_raop._tcp` TXT record

| name                | value      | description                          |
| ------------------- | ---------- | ------------------------------------ |
| txtvers             | 1          | TXT record version 1                 |
| ch                  | 2          | audio channels: stereo               |
| [cn](#audio-codecs) | 0,1,2,3    | audio codecs                         |
| et                  | 0,3,5      | supported encryption types           |
| md                  | 0,1,2      | supported metadata types             |
| pw                  | false      | does the speaker require a password? |
| sr                  | 44100      | audio sample rate: 44100 Hz          |
| ss                  | 16         | audio sample size: 16-bit            |
| tp                  | UDP        | supported transport: TCP or UDP      |
| vs                  | 130.14     | server version 130.14                |
| am                  | AppleTV2,1 | device model                         |

| FromTXTRecord       | ToDict                  | Type    | Explanation               |
| ------------------- | ----------------------- | ------- | ------------------------- |
| [cn](#audio-codecs) | compressionTypes        | BitList | Compression types         |
| da                  | rfc2617DigestAuthKey    | Boolean | RFC2617 digest auth key   |
| et                  | encryptionTypes         | BitList | Encryption types          |
| ft                  | features                | Int64   | Features                  |
| fv                  | firmwareVersion         | String  | Firmware version          |
| sf                  | systemFlags             | Int64   | System flags              |
| md                  | metadataTypes           | BitList | Metadata types            |
| am                  | deviceModel             | String  | Device model              |
| pw                  | password                | Boolean | Password                  |
| pk                  | publicKey               | String  | Public key                |
| tp                  | transportTypes          | String  | Transport types           |
| vn                  | airTunesProtocolVersion | String  | AirTunes protocol version |
| vs                  | airPlayVersion          | String  | AirPlay version           |
| ov                  | OSVersion               | String  | OS version                |
| vv                  | vodkaVersion            | Int64   | Vodka version             |


### Authentication/Encryption methods
| et  | description                |
| --- | -------------------------- |
| 0   | no encryption              |
| 1   | RSA (AirPort Express)      |
| 3   | FairPlay                   |
| 4   | MFiSAP (3rd-party devices) |
| 5   | FairPlay SAPv2.5           |

An AirPlay sender can authenticate the receiver in the following order of precedence:

- **MFi authentication** if `Authentication_8` or `SupportsUnifiedPairSetupAndMFi` are enabled;
  - If MFi authentication is enabled the sender issues a `POST /auth-setup RTSP/1.0` request.
- **FairPlay authentication** if `Authentication_4` is enabled;
  - If FairPlay authentication is enabled the sender issues a `POST /fp-setup RTSP/1.0` request.
- **RSA authentication** if `Authentication_1` bit in features is enabled.
  - Apple seems to have disabled RSA authentication, probably to block another James Laird-like attempt2 as already happened for the AirPort Express back in 2011. Enabling RSA authentication triggers a “*Not supported for AirPlay sessions*” error in the sender logs.
> If the receiver declares `SupportsHKPairingAndAccessControl`, then the authentication process is initiated after pairing is established.

When ***none*** of those bits are set, sender stops the pairing and logs an error hinting authentication is mandatory and must be enabled ([or not?](#disabled-authentication)).
#### Disabled authentication
Apple tests MFi authentication support with an **OR** condition. Moreover, the actual authentication phase begins only if bit `Authentication_8` is set. In other words when only `SupportsUnifiedPairSetupAndMFi` is enabled, 
1) we pass the authentication checks, 
2) the sender actually doesn't start any authentication setup. This sounds like a ***logic bug*** that could be fixed in the future, unless Apple keeps it as an undocumented feature… 
>Authentication can still be disabled as of `iOS 13.3`.

#### Encryption
AirPlay 2 encryption is associated to any of the `SupportsCoreUtilsPairingAndEncryption` bits in the receiver features.
The minimal set of features described in [Multi-room support](#multi-room-support) allow to disable the outer encryption layer of the protocol. The audio frames are still encrypted.

#### Disabled Encryption
Declaring only `SupportsUnifiedPairSetupAndMFi` **without** using any of the `CoreUtilsPairingAndEncryption` bits, we jump *straight* to the AirPlay 2 streaming protocol. 
> Messages at this point are exchanged in clear text.
```cpp
if !CoreUtilsPairingAndEncryption && (avoid_auth/is_apple_internal_build)
   log("*** authentication/encryption disabled ***")
```
If `CoreUtilsPairingAndEncryption` bits (38, 46, 43, 48) are **disabled** ***and*** the AirPlay sender is an Apple internal build, or `avoid_auth`, both authentication and encryption are disabled.

>The `avoid_auth` appears to be an internal condition and needs further reversing.

### Audio Formats

#### Audio Codecs
| cn  | description                  |
| --- | ---------------------------- |
| 0   | PCM                          |
| 1   | Apple Lossless (ALAC)        |
| 2   | AAC                          |
| 3   | AAC ELD (Enhanced Low Delay) |

>The following bitmask defines the possible audio formats announce by the sender with a `SETUP` request.

| Bit | Value       | Type            |
| --- | ----------- | --------------- |
| 2   | 0x4         | PCM/8000/16/1   |
| 3   | 0x8         | PCM/8000/16/2   |
| 4   | 0x10        | PCM/16000/16/1  |
| 5   | 0x20        | PCM/16000/16/2  |
| 6   | 0x40        | PCM/24000/16/1  |
| 7   | 0x80        | PCM/24000/16/2  |
| 8   | 0x100       | PCM/32000/16/1  |
| 9   | 0x200       | PCM/32000/16/2  |
| 10  | 0x400       | PCM/44100/16/1  |
| 11  | 0x800       | PCM/44100/16/2  |
| 12  | 0x1000      | PCM/44100/24/1  |
| 13  | 0x2000      | PCM/44100/24/2  |
| 14  | 0x4000      | PCM/48000/16/1  |
| 15  | 0x8000      | PCM/48000/16/2  |
| 16  | 0x10000     | PCM/48000/24/1  |
| 17  | 0x20000     | PCM/48000/24/2  |
| 18  | 0x40000     | ALAC/44100/16/2 |
| 19  | 0x80000     | ALAC/44100/24/2 |
| 20  | 0x100000    | ALAC/48000/16/2 |
| 21  | 0x200000    | ALAC/48000/24/2 |
| 22  | 0x400000    | AAC-LC/44100/2  |
| 23  | 0x800000    | AAC-LC/48000/2  |
| 24  | 0x1000000   | AAC-ELD/44100/2 |
| 25  | 0x2000000   | AAC-ELD/48000/2 |
| 26  | 0x4000000   | AAC-ELD/16000/1 |
| 27  | 0x8000000   | AAC-ELD/24000/1 |
| 28  | 0x10000000  | OPUS/16000/1    |
| 29  | 0x20000000  | OPUS/24000/1    |
| 30  | 0x40000000  | OPUS/48000/1    |
| 31  | 0x80000000  | AAC-ELD/44100/1 |
| 32  | 0x100000000 | AAC-ELD/48000/1 |


### Metadata
| md  | bit    | description |
| --- | ------ | ----------- |
| 0   | 17     | text        |
| 1   | 15     | artwork     |
| 2   | 16     | progress    |
| 50  | bplist |             |


#### CanBeRemoteControlled
`SupportsBufferedAudio` is set and `PINRequired` is not set

#### State Changes
Depending on the state of the device the mDNS record is changed to reflect this. Primarily it is the:
   - `flags`
   - `gid`,
   - `igl`,
   - `gcgl`,
   - `pgid`,
   - `pgcgl`

fields that are changed.

### `GET /info`

| key                                                | type    | value             | description                                      |
| -------------------------------------------------- | ------- | ----------------- | ------------------------------------------------ |
| PTPInfo                                            | string  |                   |                                                  |
| build                                              | string  |                   |                                                  |
| deviceID                                           | string  | 58:55:CA:1A:E2:88 | MAC address                                      |
| features                                           | integer | 14839             | features bits as decimal value                   |
| initialVolume                                      | real    |                   |                                                  |
| macAddress                                         | string  |                   |                                                  |
| firmwareBuildDate                                  | string  |                   |                                                  |
| firmwareRevision                                   | string  |                   |                                                  |
| keepAliveLowPower                                  | boolean |                   |                                                  |
| keepAliveSendStatsAsBody                           | boolean |                   |                                                  |
| manufacturer                                       | string  |                   |                                                  |
| model                                              | string  | AppleTV2,1        | device model                                     |
| name                                               | string  |                   |                                                  |
| nameIsFactoryDefault                               | boolean |                   |                                                  |
| pi                                                 | string  |                   |                                                  |
| pk                                                 | data    |                   |                                                  |
| playbackCapabilities.supportsFPSSecureStop         | boolean |                   |                                                  |
| playbackCapabilities.supportsUIForAudioOnlyContent | boolean |                   |                                                  |
| protocolVersion                                    | string  | 1                 | protocol version                                 |
| psi                                                | string  |                   |                                                  |
| senderAddress                                      | string  |                   |                                                  |
| sdk                                                | string  |                   |                                                  |
| sourceVersion                                      | string  | 120.2             | server version                                   |
| statusFlags                                        | integer | 4                 | status flags as decimal value                    |
| txtAirPlay                                         | data    | ...               | raw TXT record from AirPlay service mDNS record  |
| txtRAOP                                            | data    | ...               | raw TXT record from AirTunes service mDNS record |
| volumeControlType                                  | integer |                   |                                                  |
| vv                                                 | integer |                   |                                                  |