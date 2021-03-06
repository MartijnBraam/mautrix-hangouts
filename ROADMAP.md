# Features & roadmap
* Matrix → Hangouts
  * [ ] Message content
    * [ ] Plain text
    * [ ] Formatted messages
    * [ ] Media/files
    * [ ] Replies
  * [ ] Message redactions
  * [ ] Presence<sup>[4]</sup>
  * [ ] Typing notifications<sup>[4]</sup>
  * [ ] Read receipts<sup>[4]</sup>
  * [ ] Power level
  * [ ] Membership actions
    * [ ] Invite
    * [ ] Join
    * [ ] Leave
    * [ ] Kick
  * [ ] Room metadata changes
    * [ ] Name
    * [ ] Avatar<sup>[1]</sup>
    * [ ] Topic<sup>[1]</sup>
  * [ ] Initial room metadata
* Hangouts → Matrix
  * [ ] Message content
    * [ ] Plain text
    * [ ] Formatted messages
    * [ ] Media/files
    * [ ] Location messages
    * [ ] Replies
  * [ ] Chat types
    * [ ] Private chat
    * [ ] Group chat
    * [ ] Broadcast list<sup>[2]</sup>
  * [ ] Message deletions
  * [ ] Avatars
  * [ ] Presence
  * [ ] Typing notifications
  * [ ] Read receipts
  * [ ] Admin/superadmin status
  * [ ] Membership actions
    * [ ] Invite
    * [ ] Join
    * [ ] Leave
    * [ ] Kick
  * [ ] Group metadata changes
    * [ ] Title
    * [ ] Avatar
    * [ ] Description
  * [ ] Initial group metadata
  * [ ] User metadata changes
    * [ ] Display name<sup>[3]</sup>
    * [ ] Avatar
  * [ ] Initial user metadata
    * [ ] Display name
    * [ ] Avatar
* Misc
  * [ ] Automatic portal creation
    * [ ] At startup
    * [ ] When receiving invite<sup>[2]</sup>
    * [ ] When receiving message
  * [ ] Private chat creation by inviting Matrix puppet of WhatsApp user to new room
  * [ ] Option to use own Matrix account for messages sent from WhatsApp mobile/other web clients
  * [ ] Shared group chat portals

<sup>[1]</sup> May involve reverse-engineering the WhatsApp Web API and/or editing go-whatsapp  
<sup>[2]</sup> May already work  
<sup>[3]</sup> May not be possible  
<sup>[4]</sup> Requires [matrix-org/synapse#2954](https://github.com/matrix-org/synapse/issues/2954) or Matrix puppeting
