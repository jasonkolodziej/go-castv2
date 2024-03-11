# Chromecast Devices

`_googlecast._tcp`

| FromTXTRecord | ToDict                  | Type    | Explanation               |
|---------------|-------------------------|---------|---------------------------|
| id            | ?        | UUID? | Compression types         |
| cd            | ?    | UUID? | RFC2617 digest auth key   |
| rm            | ?         | BitList | Encryption types          |
| ve            | features ?               | Int64   | Features                  |
| ic            | systemFlags             | Int64   | System flags              |
| md            | metadataTypes           | BitList | Metadata types?            |
| fn            | fullName?             | String  | Device Name (from Google Home Setup)              |
| ca            | ?                | Int | Password                  |
| st            | ?               | String  | Public key                |
| bs            | bullSh!t          | String  | Transport types           |
| nf            | ? | Int  | AirTunes protocol version |
| rs            | resentSource?          | String  | AirPlay version           |


```
      "id=4e9843f2b74d4d380224b1a2b1c2c3e3",
      "cd=06D17599BC64D43E670559D02591BE07",
      "rm=",
      "ve=05",
      "md=Chromecast HD",
      "ic=/setup/icon.png",
      "fn=Den TV",
      "ca=465413",
      "st=1",
      "bs=FA8F05F9A2B9",
      "nf=1",
      "rs=Youtube"
```