req = require 'request'

start_t = process.hrtime()[0]
process.on "exit", ->
  console.error "Duration: ", process.hrtime()[0]-start_t, "sec"

fetchers = 800

max_uid    = 1000000
max_length = 4000
max_cnt    = 800

start_fetchers = ->
    fetch() for i in [0...fetchers]

fetch = ()->
    uids = queue.shift()
    return unless uids?

    await grab_url uids, defer err, uids, response
    unless err?
        for u in response
            {uid, sex, bdate, country} = u
            sex?=""
            bdate?=""
            country?=""
            console.log "#{uid}\t#{sex}\t#{bdate}\t#{country}"
    else
        console.error "ERROR"
        queue.push uids
    fetch()

grab_url = (uids, cb)->

    fields = "bdate,sex,country"
    url = "https://api.vk.com/method/users.get?v=3&user_ids=#{uids}&fields=#{fields}"
    await req.get url, defer err, res, data

    return cb err, uids if err
    return cb res.statusCode, uids unless res.statusCode is 200
    json = JSON.parse data
    return cb json.error, uids if json.error?
    cb null, uids, json.response


queue = []
uids = ""
cnt = 0
for uid in [0..max_uid]
    uids += uid
    if uids.length > max_length or cnt > max_cnt
        queue.push uids
        uids = ""
        cnt = 0
    else
        uids += ","
        cnt += 1

queue.push uids
start_fetchers()
