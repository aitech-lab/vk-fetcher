req = require 'request'
readline = require 'readline'
parser = require './parser'

rl = readline.createInterface
  input   : process.stdin
  output  : process.stdout
  terminal: false

queue = []
rl.on 'line', (pid)-> queue.push pid
rl.on 'close', -> start_fetchers()
    
start_fetchers = ->
    fetch() for i in [0...100]

fetch = ()->
    pid = queue.shift()
    return unless pid?
    console.error pid
    await grab_url pid, defer err, pid, uids
    unless err?
        for uid in uids
            console.log uid
    else
        queue.push pid
    fetch()

grab_url = (pid, cb)->
    url = "https://vk.com/catalog.php?selection=#{pid}"
    await req.get url, defer err, res, data

    return cb err, pid if err
    return cb res.statusCode, pid unless res.statusCode is 200

    cb null, pid, parser data
