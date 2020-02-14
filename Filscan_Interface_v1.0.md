# Filscan Interface
 

### 1.Base Struct
#### 1.1 TipSet
message <span id="TipSet"> TipSet  </span>{
    repeated FilscanBlock tipset = 1;
    string min_ticket_block = 2;  }
#### 1.2 FilscanBlock
message <span id="FilscanBlock"> FilscanBlock  </span>{
    BlockHeader block_header= 1;
    string cid = 2; 
    int64 size = 4;
    repeated string msg_cids=5 ;
     string reward =6;
    } 
####  1.3 FilscanMessage
 message <span id="FilscanMessage">FilscanMessage</span>{
    Message msg= 1;
    string cid = 2;
    int64 size = 3;
    int64 msgcreate =4;
    string method_name = 10;
     uint64 height = 5;
    string block_cid = 6;
    }
#### 1.4 FilscanActor
message <span id="FilscanActor">FilscanActor</span> {
   string address =1; 
    Actor actor = 2;
    bool is_storage_miner = 3;
    bool is_owner = 4;
    bool is_miner =5;
    bool is_wallet =6;
    uint64 Messages = 7;
}
#### 1.5 FilscanMiner
message  <span id="FilscanMiner">FilscanMiner</span> {
    string owner_address = 1;
    string peer_id = 2;
    uint64 sector_size = 3;
    int64 power = 4;
    uint64 sector_num = 5;
    int64 proving_sector_num = 6;
    int64 fault_num = 7;
}
#### 1.6 BlockHeader
message <span id="BlockHeader">BlockHeader</span> {
    string miner =1;
    repeated string tickets = 2;
    string electionProof = 3;
    repeated string parents = 4;
	 string parent_weight = 5 ;
    int64 height = 6;
	 string parent_state_root = 7;
	 string parent_message_receipts = 8;
	 string messages = 9;
    Signature bls_aggregate =10;
	 int64 timestamp = 11 ;
    Signature block_sig =12;
}
#### 1.7  Signature
message <span id="Signature">Signature</span> {
    string type =1;
    string data =2;
    }
#### 1.8  Message
message <span id="Message">Message</span>{
    string to  =1;
    string from =2;
    uint64 nonce=3;
    string value =4;
    string gasprice =5;
    string gaslimit =6 ;
    string  method =7;
    string params = 8;
}
#### 1.9  Blocktime
message  <span id="Blocktime">Blocktime</span> {
    int64 time = 1;
    double block_time = 2;  
}
#### 1.10  Blocksize
message  <span id="Blocksize">Blocksize</span> {
    int64 time = 1;
    double block_size = 2;  
}
#### 1.11  Peer
message  <span id="Peer">Peer</span> {
    string peer_id = 1;
    string miner_address =2;
    string ip =3;
    string location_cn =5;
    string location_en =6;
}
#### 1.12  PeerPoint
message  <span id="PeerPoint">PeerPoint</span> {
   string location_cn =1;
    string location_en =2;
    repeated PeerIdIp peers =3;
    double longitude =4;
    double latitude =5; 
}
#### 1.13  PeerIdIp
 message  <span id="PeerIdIp">PeerIdIp</span> {
    string peer_id = 1;
    string ip =3;
    }
#### 1.14  TotalPowerGraphical
 message  <span id="TotalPowerGraphical">TotalPowerGraphical</span> {
    int64 time = 1;
    int64 power = 2;
    }
#### 1.15  MinerState
message  <span id="MinerState">MinerState</span> {
    string address = 1;
    string power = 2;
    string power_percent = 3;
    string peer_id = 4;
} 

#### 1.16  FutureBlockRewardData
message  <span id="FutureBlockRewardData">FutureBlockRewardData</span> {
    uint64 time = 1;
    string block_rewards = 2;
    string vested_rewards = 3;
} 

#### 1.17  CBRORData
message  <span id="CBRORData">CBRORData</span> {
    uint64 time_start = 1;
    uint64 time_end = 2;
    string blocks_reward = 3;
} 

#### 1.18  FiloutstandData
message  <span id="FiloutstandData">FiloutstandData</span> {
    uint64 time_start = 1;
    uint64 time_end = 2;
    string blocks_reward = 3;
    }
  
#### 1.19 MinerInfo
message <span id="MinerInfo">MinerInfo</span> {
    string increased_power = 1;
    uint64 increased_block = 2;
    string power_percent = 3;
    string block_percent = 4;
    string mining_efficiency = 5;
    string storage_rate = 6;
    string miner = 7;
    string peer_id = 8;
}
#### 1.20 Actor 
message <span id="Actor">Actor</span>{
    string code =1;
    string head=2;
    uint64 nonce=3;
    string Balance =4;
}


---
###  2.Public Response Parameters 

message InterfaceResp {
    message Data {
    }
    common.Result res = 1;
    Data data = 2;
};
message Result {
  //  3-success，5-failure
  int32 code = 1; 
  // response message
  string msg = 2; 
  }
 
---
###  3 Home Page
#### 3.1 SearchIndex

Fuction
> search key words

 URI
> v0/filscan/SearchIndex

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----                               |
|key    |ture    |string|index keyword                           |
|filter    |ture    |int64|1-address , 2-message_ID , 3-Height , 4-block_hash , 5-peer_id  |
 Response:
 
|Fields|Type|Remark                              |
|:-----   |:------|:-----------------------------   |
|model_flag   |string    |maybe:actor,message_ID ,Height , block_hash , peer_id| 

#### 3.2 LatestBlock
 Fuction
> the latest block

 URI
> v0/filscan/LatestBlock

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----   |
|num    |ture    |int64|latest number |

 Response:
 
|Fields|Type|Remark                              |
|:-----   |:------|:-----------------------------   |
|block_header   |[]FilscanBlock    |BlockArr| 

#### 3.3 LatestMsg
 Fuction
> the latest Msg

 URI
> v0/filscan/LatestMsg

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----   |
|num    |ture    |int64|latest num |

 Response:
 
|Fields|Type|Remark                              |
|:-----   |:------|:-----------------------------   |
|msg   |[]FilscanMessage|MessageArr| 

#### 3.4 BaseInformation
 Fuction
> Basic Information

 URI
> v0/filscan/BaseInformation

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----   |

 Response:
 
|Fields|Type|Remark                              |
|:-----   |:------|:-----------------------------   |
|tipset_height   |uint64| | 
|block_reward   |float64| | 
|avg_message_size   |float64| | 
|avg_gas_price   |float64| | 
|avg_messages_tipset   |float64| | 
|pledge_collateral   |string| |  

#### 3.5 BlocktimeGraphical
 Fuction
> Chart of Block's timestamp

 URI
> v0/filscan/BlocktimeGraphical

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----   |
|start_time  |ture    |int64|  |
|end_time |ture    |int64|  |


 Response:
 
|Fields|Type|Remark                              |
|:-----   |:------|:-----------------------------   |
|min   |float64| | 
|max   |float64| | 
|avg_blocktime   |float64| | 
|data   |[]Blocktime |  

#### 3.6 AvgBlockheaderSizeGraphical
 Fuction
> Chart of average block size

 URI
> v0/filscan/BlocktimeGraphical

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----   |
|start_time  |ture    |int64|  |
|end_time |ture    |int64|  |


 Response:
 
|Fields|Type|Remark                              |
|:-----   |:------|:-----------------------------   |
|min   |float64| | 
|max   |float64| | 
|avg_blocksize   |float64| | 
|data   |[]Blocksize| |

#### 3.7 TotalPowerGraphical
 Fuction
> Chart of TotalPower in last 24h

 URI
> v0/filscan/TotalPowerGraphical

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|time    |false    |int64|time end 默认now  | 

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|storage_capacity |float64 |  | 
|data        |[]TotalPowerGraphical |  |  
 
---
###  4 Messages
#### 4.1 BlockMessages
 Fuction
> Search Bar

 URI
> v0/filscan/messages/BlockMessages

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  |
|block_cid    |false  |string|return all msg if empty  |
|begindex    |ture    |int64|start index |
|count    |ture       |int64|return count |
|method    |false     |string|return all method's msg if empty |

 Response:
 
|Fields|Type|Remark                              |
|:-----   |:------|:-----------------------------   |
|msgs    |[]FilscanMessage |MessageArr|
|total    |int64 |page count|

#### 4.2 MessagesMethods
 Fuction
> Get msg method list

 URI
> v0/filscan/messages/MessagesMethods

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  |
|cids    |false  |[]string|return all msg method list if empty| 

 Response:
 
|Fields|Type|Remark                              |
|:-----   |:------|:-----------------------------   |
|method    |[]string | | 
#### 4.3 MessageDetails
 Fuction
> Msg Details

 URI
> v0/filscan/messages/MessageDetails

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  |
|cids    |false  |[]string|return all msg method list if empty| 

 Response:
 
|Fields|Type|Remark                              |
|:-----   |:------|:-----------------------------   |
|msg    |FilscanMessage  | | 

#### 4.4 MessageByAddress
 Fuction
> Get address's msg

 URI
> v0/filscan/messages/MessageByAddress

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  |
|address    |true  |string|address in message| 
|begindex    |ture    |int64|start index |
|count    |ture       |int64|return count |
|method    |false  |string|return all method's msg if empty| 
|from_to    |false  |string|value of "from","to",""|  

 Response:
 
|Fields|Type|Remark                              |
|:-----   |:------|:-----------------------------   |
|data    |[]FilscanMessage | MessageArr |
|total    |int64 |page count |
###  5 tipset
#### 5.1 BlockByHeight
 Fuction
> Get block by height

 URI
> v0/filscan/tipset/BlockByHeight

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|height    |ture    |uint64|  | 

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|blocks   |[]FilscanBlock |  | 

#### 5.2 BlockByMiner
 Fuction
> Search blocks by miner address

 URI
> v0/filscan/tipset/BlockByMiner

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|miners    |ture    |[]string|   | 
|begindex    |ture    |int64|start index |
|count    |ture       |int64|return count |

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|blocks   |[]FilscanBlock |  |
|total    |int64 |page count |
|total_fil    |string |  |

#### 5.3 BlockByCid
 Fuction
> Get block by blockCid

 URI
> v0/filscan/tipset/BlockByCid

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|cid    |ture    |string|   | 

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|blocks   |FilscanBlock |  |

#### 5.4 TipSetTree
 Fuction
>  Get chain struct

 URI
> v0/filscan/tipset/TipSetTreeReq

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|count     |ture    |int64| count  | 
|end_height |ture    |int64| height  | 

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|tipsets   |[]TipSet |  | 

#### 5.5 BlockConfirmCount
 Fuction
> Get count of block which confirmed 

 URI
> v0/filscan/tipset/BlockConfirmCount

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|cid     |ture    |string|    |  

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|count   |string |  | 

----
###  6 peer
#### 6.1 PeerById
 Fuction
> Get peer information by id

 URI
> v0/filscan/peer/PeerById

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|peer_id    |ture    |string|  | 

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|peer   |Peer |  | 

#### 6.2 ActivePeerCount
 Fuction
> Count of active peers

 URI
> v0/filscan/peer/PeerById

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  |  

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|count   |string |  | 

#### 6.3 PeerMap
 Fuction
> Peer map

 URI
> v0/filscan/peer/PeerMap

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  |  

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|peer_point   |[]PeerPoint |  | 
---

###  7 account
#### 7.1 ActorById
 Fuction
> Get actor's detailed information by id

 URI
> v0/filscan/account/ActorById

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|actor_id    |ture    |string|  | 

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|data   |FilscanActor |  | 
|work_list   |[]string |  | 
|miner   |FilscanMiner |  | 

#### 7.2 AccountList
 Fuction
> Get account list

 URI
> v0/filscan/account/AccountList

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|begindex    |ture    |int64|start index |
|count    |ture       |int64|return count |

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|accounts   |[]FilscanActor |  | 
|total   |uint64 |  | 
|total_fil   |string |  | 

#### 7.3 WorkListByAddress
 Fuction
> WorkList By Owner Address

 URI
> v0/filscan/account/WorkListByAddress

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|address    |ture    |string| | 

 Response:

|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|address   |[]string ||  
---
###  8 mining
#### 8.1 MinerList
 Fuction
> Get power changed in period.

 URI
> /v0/filscan/mining/MinerList

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|time_start    |ture    |int64| start time | 
|time_at    |ture    |int64| end time, default is now | 
|offset    |ture    |int64| start index | 
|limit    |ture    |int64| return count | 
|sort|true|string|fields to be sorted, value:'power','block','power_rate','mining_efficiency','power' by default|
|sort_type|true|int|sorting type, ASC if value >= 0 , DESC if value < 0 , '-1' by default |

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|total_increased_power |string| power growth in period | 
|total_increased_block |string|  block count in period  | 
|miner_count |uint64    | count of miner | 
|miners|[]MinerInfo | miner info  |  

#### 8.2 MinerSearch
 Fuction
> MinerSearch 

 URI
> v0/filscan/mining/MinerSearch

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|miner    |ture    |string|miner id |  

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|miners |[]MinerState |  |  

#### 8.3 ActiveStorageMinerCountAtTime
 Fuction
> Get miner count by time

 URI
> v0/filscan/mining/ActiveStorageMinerCountAtTime

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|time_at    |ture    |uint64|end time|  
|time_diff    |ture    |uint64|period |  
|repeate_time    |ture    |uint64| repeat times|  

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|miners |[]MinerState |  |  

#### 8.4 MinerPowerAtTime
 Fuction
> MinerPowerAtTime

 URI
> v0/filscan/mining/MinerPowerAtTime

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|time_at    |ture   | uint64|end time|
|time_diff    |ture | uint64|period| 
|repeate_time |ture | uint64|repeat time| 
|miners    |ture    |[]string| miner array |  

 Response:  //todo
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |

#### 8.5 TopnPowerMiners
 Fuction
> Top miners by power

 URI
> v0/filscan/mining/TopnPowerMiners

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|time_at    |ture   | uint64|end time|   
|offset    |ture    |int64| start index | 
|limit    |ture    |int64| return count | 

 Response:  //todo
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   |
|total_miner_count |uint64 |   |  
|miners |[]MinerState |  |  
---
###  9 token
#### 9.1 FilNeworkBlockReward
 Fuction
> FilNewworkBlockReward

 URI
> v0/filscan/mining/FilNewworkBlockReward

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|time_diff    |ture | uint64|period |  
|repeate |ture | uint64| repeat times |  

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   | 
|data        |[]FutureBlockRewardData |  |  

#### 9.2 CumulativeBlockRewardsOverTime
 Fuction
> CumulativeBlockRewardsOverTime 

 URI
> v0/filscan/mining/CumulativeBlockRewardsOverTime

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|time_diff    |ture | uint64|period |  
|repeate |ture | uint64| repeat times | 
|time_start |ture | uint64| start time |   

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   | 
|data        |[]CBRORData |  |  

#### 9.3 FilOutStanding
 Fuction
> Outstanding Fil amount

 URI
> v0/filscan/mining/FilOutStanding

Format
> JSON

 HTTP Method
> Post

 Request:
 
|Params|Required|Type|Remark|
|:-----  |:-------|:-----|-----  | 
|time_diff    |ture | uint64|period |  
|repeate |ture | uint64| repeat time | 
|time_start |ture | uint64| start time |   

 Response:
 
|Fields|Type|Remark |
|:-----   |:------|:-----------------------------   | 
|data        |[]FiloutstandData |  |  