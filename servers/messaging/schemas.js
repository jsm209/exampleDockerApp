const Schema = require("mongoose").Schema;

/*
const cakeSchema = new Schema({
    whoFor: {type: String, required: true, unique: false},
    createdAt: {type: Date, required: true},
    numCandles: Number
});
*/

const channelSchema = new Schema({
    name: {type: String, required: true, unique: false},
    description: {type: String, required: false, unique: false},
    private: {type: Boolean, required: false, default: false},
    members: {type: [{
        id: Number,
        Email: String,
        PassHash: String,
        UserName: String,
        FirstName: String,
        LastName: String,
        PhotoURL: String
    }]},
    createdAt: {type: Date, required: true},
    creator: {type: {
        id: Number,
        Email: String,
        PassHash: String,
        UserName: String,
        FirstName: String,
        LastName: String,
        PhotoURL: String
    }, required: true}, 
    editedAt: Date
});

const messageSchema = new Schema({
    channelID: {type: Number, required: true},
    body: String,
    createdAt: {type: Date, required: true},
    creator: {type: {
        id: Number,
        Email: String,
        PassHash: String,
        UserName: String,
        FirstName: String,
        LastName: String,
        PhotoURL: String
    }, required: true}, 
    editedAt: Date
});

module.exports = {channelSchema, messageSchema}

//module.exports = { cakeSchema }