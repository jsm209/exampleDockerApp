"use strict";

// Require express, morgan, and mongoose packages
const express = require("express");
const morgan = require("morgan");
const mongoose = require("mongoose"); // used by nodeJS to talk to database
const { messageSchema, channelSchema } = require('./schemas');

// also require rabbitMQ to send messages to queue
var amqp = require('amqplib/callback_api');

const mongoEndpoint = "mongodb://mongodb:27017/messaging"

// get ADDR environmental variable
const addr = process.env.ADDR || ":80";

// split host and port
const [host, port] = addr.split(":");

//const port = process.env.PORT;
//const instanceName = process.env.NAME;


// MODELS
const Message = mongoose.model("Message", messageSchema);
const Channel = mongoose.model("Channel", channelSchema);


// create a new express application
const app = express(); 

//add JSON request body parsing middleware
app.use(express.json());

// connects to mongo database at endpoint
const connect = () => {
    mongoose.connect(mongoEndpoint);
}

// Connecting mongoose to our mongodb
connect();
mongoose.connection.on('error', console.error)
    .once('open', main);
/*
    .on('disconnected', connect)
*/

let rabbitmqChannel; // tracks our rabbit channel
const getRabbitChannel = () => {
    return rabbitmqChannel;
}

async function main() {
    // connect to the rabbitMQ server, create a channel, and declare a queue...
    amqp.connect('amqp://guest:guest@rabbitmq:5672/', function(error0, connection) {
        if (error0) {
            throw error0;
        }
        connection.createChannel(function(error1, channel) {
            if (error1) {
                throw error1;
            }
    
            var queue = "messages";
            //var msg = 'Hello World!';
    
            channel.assertQueue(queue, {
                durable: true,
                autoDelete: false 
                // the rest of the options default to false which matches my queue
                
            });

            // stores the channel reference so we can access it globally
            rabbitmqChannel = channel;

            //channel.sendToQueue(queue, Buffer.from(msg));
            //console.log(" [x] Sent %s", msg);
    
            // insures that we only start listening once amqp is connected
            console.log("connected successfully to rabbitMQ");
            app.listen(port, "", () => {
                console.log("server is listening on " + port);
            });
        });
    });
}



//////////////////
// ALL HANDLERS //
//////////////////

// TODO: check is X-User header is in request...

////////////////////
// "/v1/channels" //
////////////////////
app.get("/v1/channels", async (req, res) => {
    checkForAuthenticatedUser(req, res);
    try {
        // get public channels
        const publicchannels = await Channel.find( {private: false} );

        // then get the private channels where the user is a member.
        let user = req.headers["X-User"];
        const privatechannels = await Channel.find( {members: { $elemMatch: { UserName: user.Username } } } );

        res.writeHead(200, {'Content-Type': 'application/json'});
        res.json(publicchannels + privatechannels);
    } catch (err) {
        res.status(500).send("There was an issue getting channels.");
    }
});

app.post("/v1/channels", (req, res) => {
    checkForAuthenticatedUser(req, res);
    const {name, description} = req.body;
    const createdAt = new Date();
    const creator = req.headers["X-User"];
    const channel = {
        name: name,
        description: description,
        private: false,
        members: [],
        createdAt: createdAt,
        creator: creator,
        editedAt: null
    };

    const query = new Channel(channel);

    query.save((err, newChannel) => {
        if (err) {
            res.status(500).send("Unable to create a channel.");
            return;
        }

        // send event object to rabbitmq queue
        let ch = getRabbitChannel();
        let users = [];
        if (channel.private) {
            users = channel.members;
        }
        ch.sendToQueue("messages", Buffer.from(JSON.stringify(
            {
                type: "channel-new",
                channel: channel,
                userIDs: users
            }
        )));

        res.status(201).json(newChannel);

    });
});

////////////////////////////////
// "/v1/channels/{channelID}" //
////////////////////////////////
app.get("/v1/channels/:channelID", async (req, res) => {
    checkForAuthenticatedUser(req, res);
    try {
        // find the channel by the id
        const channel = await Channel.find( {_id: req.params.channelID} )

        // if private, see if the user is contained in the memebers.
        if (channel.private) { 
            let user = req.headers["X-User"];
            let isMember = checkIfUserIsMember(user, channel);
            if (!isMember) {
                res.status(403).send("You don't have permission to view that channel.");
                return;
            }
        }

        // use before parameter if it exists
        let beforeDate = null;
        if (req.query.before) {
            beforeMessage = Message.find( {channelID: req.query.before} );
            beforeDate = beforeMessage.createdAt;
        }

        // if we're here, then everything is find, send 100 recent messages

        // if QSP before provided, return the most recent 100 messages 
        // in the specified channel with message IDs less than the message ID in that query string parameter.
        const messages = null;
        if (beforeDate != null) {
            messages = await Message.find( {channelID: channel._id, "createdAt": {"$gte": beforeDate, "$lt": Date.now()}} ).sort({'createdAt': -1}).limit(100);
        } else {
            messages = await Message.find( {channelID: channel._id} ).sort({'createdAt': -1}).limit(100);
        }
        
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.json(messages);
    } catch (err) {
        res.status(500).send("There was an issue getting channel messages.");
    }
});

app.post("/v1/channels/:channelID", async (req, res) => {
    checkForAuthenticatedUser(req, res);

    try {
        // find the channel by the id
        const channel = await Channel.find( {_id: req.params.channelID} )

        // if private, see if the user is contained in the memebers.
        if (channel.private) { 
            let user = req.headers["X-User"];
            let isMember = checkIfUserIsMember(user, channel);
            if (!isMember) {
                res.status(403).send("You don't have permission to view that channel.");
                return;
            }
        }

        // otherwise we're fine, so create a message
        const body = req.body;
        const createdAt = new Date();
        const creator = req.headers["X-User"];
        const message = {
            channelID: channel._id,
            body: body,
            createdAt: createdAt,
            creator: creator,
            editedAt: null
        };

        const query = new Message(message);
        query.save((err, newMessage) => {
            if (err) {
                res.status(500).send("Unable to create message.");
                return
            }

            // send event object to rabbitmq queue
            let ch = getRabbitChannel();
            let users = [];
            if (channel.private) {
                users = channel.members;
            }
            ch.sendToQueue("messages", Buffer.from(JSON.stringify(
                {
                    type: "message-new",
                    message: message,
                    userIDs: users
                }
            )));

            res.status(201).json(newMessage)
        })
    } catch (err) {
        res.status(500).send("There was an issue getting channel messages.");
    }
});

app.patch("/v1/channels/:channelID", async (req, res) => {
    checkForAuthenticatedUser(req, res);
    try {
        const channel = await Channel.find( {_id: req.params.channelID} )
        const creator = req.headers["X-User"];
        if (channel.creator.id != creator.ID) {
            res.status(403).send("You can't update that channel.");
        } else {
            // update the channel with the name and description
            let newName = req.body.name;
            let newDescription = req.body.description;
            Channel.findOneAndUpdate(
                {_id: req.params.channelID},
                {"$set": {"name": newName, "description": newDescription}},
                function(err, result) {
                    if (err) {
                        res.send(err);
                    } else {
                        // even though the rows update in the mongoDB,
                        // we just need to modify channel so we can output it correctly

                        channel.name = newName;
                        channel.description = newDescription

                        res.writeHead(200, {'Content-Type': 'application/json'});
                        res.json(channel);
                    } 
                }
            )

            // send event object to rabbitmq queue
            let ch = getRabbitChannel();
            let users = [];
            if (channel.private) {
                users = channel.members;
            }
            ch.sendToQueue("messages", Buffer.from(JSON.stringify(
                {
                    type: "channel-update",
                    channel: channel,
                    userIDs: users
                }
            )));
        }
    } catch (err) {
        res.status(500).send("There was an issue getting the channel.");
    }
});

app.delete("/v1/channels/:channelID", async (req, res) => {
    checkForAuthenticatedUser(req, res);
    try {
        const channel = await Channel.find( {_id: req.params.channelID} )
        const creator = req.headers["X-User"];
        if (channel.creator.id != creator.ID) {
            res.status(403).send("You can't delete that channel.");
        } else {
            // delete the channel
            Channel.findOneAndDelete( {_id: req.params.channelID} )

            // delete all the messages associated with that channel
            Message.deleteMany( {channelID: req.params.channelID} )
            res.status(201).send("Deleted channel and associated messages.");

            // send event object to rabbitmq queue
            let ch = getRabbitChannel();
            let users = [];
            if (channel.private) {
                users = channel.members;
            }
            ch.sendToQueue("messages", Buffer.from(JSON.stringify(
                {
                    type: "channel-delete",
                    channelID: req.params.channelID,
                    userIDs: users
                }
            )));
        }
    } catch (err) {
        res.status(500).send("There was an issue getting the channel.");
    }
});

////////////////////////////////////////
// "/v1/channels/{channelID}/members" //
////////////////////////////////////////
app.post("/v1/channels/:channelID/members", async (req, res) => {
    checkForAuthenticatedUser();
    const user = req.headers["X-User"];
    try {
        const channel = await Channel.find( {_id: req.params.channelID} )
        let isCreator = checkIfUserIsCreator(user, channel);
        if (isCreator) {
            let currentMembers = channel.members;
            let suppliedUser = req.body;
            currentMembers.push(suppliedUser);

            Channel.findByIdAndUpdate( 
                {_id: req.params.channelID},
                {members: currentMembers},
                function(err, result) {
                    if (err) {
                        res.send(err);
                    } else {
                        res.status(201).send("User, " + suppliedUser.ID + " was successfully added to members")
                    } 
                }
            );
        } else {
            res.status(403).send("You are not the creator of this channel.");
        }
    } catch (err) {
        res.status(500).send("There was an issue getting the channel.");
    }
});

app.delete("/v1/channels/:channelID/members", async (req, res) => {
    checkForAuthenticatedUser(req, res);
    try {
        const channel = await Channel.find( {_id: req.params.channelID} )
        const creator = req.headers["X-User"];
        if (channel.creator.id != creator.ID) {
            res.status(403).send("You can't delete members from this channel.");
        } else {
            // get the current collection of members
            let currentMembers = channel.members;
            let suppliedUser = req.body;

            // remove the provided user
            for (let i = 0; i < currentMembers.length; i++) {
                if (currentMembers[i].id == suppliedUser.ID) {
                    currentMembers.splice(index, 1);
                }
            }

            // update the channel model
            Channel.findByIdAndUpdate(
                {_id: req.params.channelID},
                {members: currentMembers},
                function(err, result) {
                    if (err) {
                        res.send(err);
                    } else {
                        res.status(201).send("User, " + suppliedUser.ID + " was successfully deleted from members")
                    }
                }
            );
        }
    } catch (err) {
        res.status(500).send("There was an issue getting channel messages.");
    }
});

////////////////////////////////
// "/v1/messages/{messageID}" //
////////////////////////////////

app.patch("/v1/messages/:messageID", async (req, res) => {
    checkForAuthenticatedUser(req, res);
    try {
        const message = await Message.find( {_id: req.params.channelID} )
        const creator = req.headers["X-User"];
        if (message.creator.id != creator.ID) {
            res.status(403).send("You can't delete a message you don't own.");
        } else {
            Message.findByIdAndUpdate(
                {_id: req.params.messageID},
                {body: req.body},
                function(err, result) {
                    if (err) {
                        res.send(err);
                    } else {
                        message.body = req.body;
                        res.writeHead(200, {'Content-Type': 'application/json'});
                        res.json(message);
                    } 
                }
            )

            // send event object to rabbitmq queue
            let ch = getRabbitChannel();
            let users = [];
            if (channel.private) {
                users = channel.members;
            }
            ch.sendToQueue("messages", Buffer.from(JSON.stringify(
                {
                    type: "message-update",
                    message: message,
                    userIDs: users
                }
            )));
        }
    } catch (err) {
        res.status(500).send("There was an issue getting channel messages.");
    }
});

app.delete("/v1/messages/:messageID", async (req, res) => {
    checkForAuthenticatedUser(req, res);
    try {
        const message = await Message.find( {_id: req.params.messageID} )
        const creator = req.headers["X-User"];
        if (message.creator.id != creator.ID) {
            res.status(403).send("You can't delete a message you don't own.");
        } else {
            // delete the message with that id
            Message.findOneAndDelete( {_id: req.params.messageID} )
        
            // send event object to rabbitmq queue
            let ch = getRabbitChannel();
            let users = [];
            if (channel.private) {
                users = channel.members;
            }
            ch.sendToQueue("messages", Buffer.from(JSON.stringify(
                {
                    type: "message-delete",
                    messageID: req.params.messageID,
                    userIDs: users
                }
            )));

            res.status(200).send("Message successfully deleted");
        }
    } catch (err) {
        res.status(500).send("There was an issue getting channel messages.");
    }
});



////////////////////////

// post request endpoint
app.post("/v1/cake", (req, res) => {
    const {whoFor, numCandles} = req.body;
    if (!whoFor) {
        res.status(400).send("Must provide cake recipient");
        return;
    }

    if (typeof numCandles !== "number") {
        res.status(400).send("numCandles must be number.");
        return;
    }

    // creates a new cake with the valid request body
    const createdAt = new Date();
    const cake = {
        whoFor,
        numCandles,
        createdAt
    };

    // capital cake references earlier mongoose.model, defined with the schema
    const query = new Cake(cake); 
    
    // saves it to mongo DB.
    query.save((err, newCake) => {
        if (err) {
            res.status(500).send("Unable to create a cake");
            return;
        }

        res.status(201).json(newCake);
    }); 
})

// given the response and request headers,
// will check if there is an X-User present.
// If not, will respond with a 401 error.
let checkForAuthenticatedUser = (req, res) => {
    if (req.headers["X-User"] == null) {
        res.status(401).send("Unable to find authenticated user.");
    }
}

// Given a channel model, will check if the given
// user is contained in the users.
// returns true/false accordingly
function checkIfUserIsMember(user, channel) {
    for (let i = 0; i < channel.members.length; i++) {
        if (channel.members[i].UserName == user.UserName) {
            return true;
        }
    }
    return false;
}

// Given a channel model, will check if the given
// user is the creator of the channel.
// returns true/false accordingly
function checkIfUserIsCreator(user, channel) {
    return (channel.creator.UserName == user.UserName);
}

// boilerplate from mongodb website
// connect to MongoClient
/*
var MongoClient = require('mongodb').MongoClient;

MongoClient.connect("mongodb://localhost:27017/exampleDb", function(err, db) {
  test.equal(null, err);
  test.ok(db != null);

  db.collection("replicaset_mongo_client_collection").update({a:1}, {b:1}, {upsert:true}, function(err, result) {
    test.equal(null, err);
    test.equal(1, result);

    db.close();
    test.done();
  });
});
*/

// boilerplate from dr.sterns notes..?
/*
app.get("/v1/chat", (req, res) => {
	res.json({
		"message": "Hello from " + instanceName 
	});
});

app.listen(port, "", () => {
        //callback is executed once server is listening
        console.log(`server is listening at http://:${port}...`);
	console.log("port : " + port);
	console.log("host : " + instanceName);
});
*/
