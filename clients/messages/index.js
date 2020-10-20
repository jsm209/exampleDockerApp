console.log("is working");
console.log("updated.");


window.onload=function() {
    // get the current auth token
    let token = window.localStorage.getItem("authToken");

    // get a websocket
    this.socket = new WebSocket("ws://localhost:5000/ws?auth=" + token);

    // attaching listeners
    document.getElementById("signin-submit").addEventListener("click", signin);
    document.getElementById("loggedin-signout").addEventListener("click", signout);
    document.getElementById("loggedin-search-input").addEventListener("change", search);
    
    document.getElementById("loggedin-createchannel").addEventListener("click", createchannel);
    document.getElementById("loggedin-deletechannel").addEventListener("click", deletechannel);
    document.getElementById("loggedin-send").addEventListener("change", send);

    this.areWeAuthorized();
}

function createchannel() {
    let data = {
        name: document.getElementById("loggedin-channelname-input").value,
        description: document.getElementById("loggedin-channeldescription-input").value
    }

    let query = document.getElementById("loggedin-channel-input").value;
    let url = "/v1/channels" + query;
    let params = {
        headers: {
            "content-type": "application/json; charset=UTF-8",
            "Authorization": window.localStorage.getItem("authToken")
        },
        body: JSON.stringify(data),
        method: "POST"
    }
    fetch(url, params)
        .then(response => {
            if (response.status < 300) {
                window.localStorage.setItem("authToken", response.headers.get("Authorization"));
            }
            return response;
        })
        .then(response => {
            console.log("created channel: " + document.getElementById("loggedin-channelname-input").value);
            let channel = JSON.parse(response);

            // stores channel id in local storage
            window.localStorage.setItem(document.getElementById("loggedin-channelname-input").value, channel.id);
        })
        .catch(err => {
            document.getElementById("error").textContent = err;
        })
}

//fix later
function deletechannel() {
    let data = {
        name: document.getElementById("loggedin-channelname-input").value,
        description: document.getElementById("loggedin-channeldescription-input").value
    }

    let query = window.localStorage.getItem(document.getElementById("loggedin-channel-input").value);
    let url = "/v1/channels/" + query;
    let params = {
        headers: {
            "content-type": "application/json; charset=UTF-8",
            "Authorization": window.localStorage.getItem("authToken")
        },
        body: JSON.stringify(data),
        method: "DELETE"
    }
    fetch(url, params)
        .then(response => {
            if (response.status < 300) {
                window.localStorage.setItem("authToken", response.headers.get("Authorization"));
            }
            return response;
        })
        .then(response => {
            console.log("deleted channel: " + document.getElementById("loggedin-channelname-input").value);
        })
        .catch(err => {
            document.getElementById("error").textContent = err;
        })
}

function send() {
    let query = window.localStorage.getItem(document.getElementById("loggedin-channel-input").value);
    let url = "/v1/channels" + query;
    let params = {
        headers: {
            "content-type": "application/json; charset=UTF-8",
            "Authorization": window.localStorage.getItem("authToken")
        },
        body: document.getElementById("loggedin-message-input"),
        method: "POST"
    }
    fetch(url, params)
        .then(response => {
            if (response.status < 300) {
                window.localStorage.setItem("authToken", response.headers.get("Authorization"));
            }
            return response;
        })
        .then(response => {
            console.log("sent message: " + document.getElementById("loggedin-message-input").value);
            let message = JSON.parse(response);

            // stores channel id in local storage
            let div = document.createElement("div");
            let p = document.createElement("p");
            let input = document.createElement("input");
            let editButton = document.createElement("button");
            let deleteButton = document.createElement("button");
        
            p.textContent = response.body;
            input.placeholder = "edit message text"
            editButton.textContent = "Edit";
            editButton.addEventListener("click", () => {
                // update message
                let url = "/v1/messages/" + message.id;
                let params = {
                    headers: {
                        "content-type": "application/json; charset=UTF-8",
                        "Authorization": window.localStorage.getItem("authToken")
                    },
                    body: input.value,
                    method: "PATCH"
                }
                fetch(url, params)
                .then(response => {
                    console.log("updated message to: " + input.value);
                })
                .catch(err => {
                    document.getElementById("error").textContent = err;
                })

            })
            deleteButton.textContent = "Delete";
            deleteButton.addEventListener("click", () => {
                // remove message
                let url = "/v1/messages/" + message.id;
                let params = {
                    headers: {
                        "content-type": "application/json; charset=UTF-8",
                        "Authorization": window.localStorage.getItem("authToken")
                    },
                    method: "DELETE"
                }
                fetch(url, params)
                .then(response => {
                    console.log("deleted message: " + p.textContent);
                })
                .catch(err => {
                    document.getElementById("error").textContent = err;
                })

                // remove from dom
                div.remove();
                p.remove();
                editButton.remove();
                deleteButton.remove();

            })

            div.appendChild(p);
            div.appendChild(editButton);
            div.appendChild(deleteButton);
            document.getElementById("loggedin-messagelog").appendChild(div);

        })
        .catch(err => {
            document.getElementById("error").textContent = err;
        })
}

function search() {
    let query = document.getElementById("loggedin-search-input").value;
    let url = "/v1/users?q=" + query;
    let params = {
        headers: {
            "content-type": "application/json; charset=UTF-8",
            "Authorization": window.localStorage.getItem("authToken")
        },
        method: "GET"
    }
    fetch(url, params)
    .then(response => {
        if (response.status < 300) {
            return response;
        }
    })
    .then(JSON.parse)
    .then(response => {
        // Clear the results table,
        let table = document.getElementById("loggedin-search-results");
        table.getElementsByTagName("tbody")[0].innerHTML = "";

        // Then loop through all responses and add them to the results table
        for (let i = 0; i < response.length; i++) {
            let tr = document.createElement("tr");
            let usernameData = document.createElement("td");
            let firstnameData = document.createElement("td");
            let lastnameData = document.createElement("td");
            let photoData = document.createElement("td");
            let image = document.createElement("image");

            usernameData.textContent = response[i].UserName;
            firstnameData.textContent = response[i].FirstName;
            lastnameData.textContent = response[i].LastName;
            image.src = response[i].PhotoURL + "?s=100"
            photoData.appendChild(image);

            tr.appendChild(usernameData);
            tr.appendChild(firstnameData);
            tr.appendChild(lastnameData);
            tr.appendChild(photoData);

            table.appendChild(tr);
        }
    })
    .catch(err => {
        document.getElementById("error").textContent = err;
    })
}

function signin() {

    // build credentials object
    let credentials = {
        Email: document.getElementById("signin-email").value,
        Password: document.getElementById("signin-password").value
    }

    let url = "/v1/sessions";
    let params = {
        headers: {
            "content-type": "application/json; charset=UTF-8"
        },
        body: JSON.stringify(credentials),
        method: "POST"
    }

    fetch(url, params)
        .then(response => {
            if (response.status < 300) {
                window.localStorage.setItem("authToken", response.headers.get("Authorization"));
            }
            return response;
        })
        .then(response => {
            // will get the current user and switch screens
            myUser = getCurrentUser();
            document.getElementById("loggedin-firstname").textContent = myUser.FirstName;
            document.getElementById("loggedin-lastname").textContent = myUser.LastName;
        })
        .catch(err => {
            document.getElementById("error").textContent = err;
        })
}

function signup() {

    // build new user object
    let newUser = {
        Email: document.getElementById("signup-email").value,
        Password: document.getElementById("signup-password").value,
        PasswordConf: document.getElementById("signup-passwordconfirm").value,
        UserName: document.getElementById("signup-username").value,
        FirstName: document.getElementById("signup-firstname").value,
        LastName: document.getElementById("signup-lastname").value
    }

    let url = "/v1/users";
    let params = {
        headers: {
            "content-type": "application/json; charset=UTF-8"
        },
        body: JSON.stringify(newUser),
        method: "POST"
    }

    fetch(url, params)
    .then(response => {
        if (response.status < 300) {
            window.localStorage.setItem("authToken", response.headers.get("Authorization"));
        }
    })
    .then(response => {
        // will get the current user and switch screens
        myUser = getCurrentUser();
        document.getElementById("loggedin-firstname").textContent = myUser.FirstName;
        document.getElementById("loggedin-lastname").textContent = myUser.LastName;
    })
    .catch(err => {
        document.getElementById("error").textContent = err;
    })

}

function signout() {
    let url = "v1/sessions/mine";
    let params = {
        headers: {
            "content-type": "application/json; charset=UTF-8",
            "Authorization": window.localStorage.getItem("authToken")
        },
        method: "DELETE"
    }

    fetch(url, params)
    .then(response => {
        if (response.status < 300) {
            window.localStorage.clear();

            // confirms token is deleted, and returns to the signin/signup page
            this.areWeAuthorized();
        }
    })
    .catch(err => {
        document.getElementById("error").textContent = err;
    })
}

function update(firstname, lastname) {
    let user = getCurrentUser();

    let updates = {
        FirstName: firstname,
        LastName: lastname
    }

    let url = "v1/users/" + user.ID
    let params = {
        headers: {
            "content-type": "application/json; charset=UTF-8",
            "Authorization": window.localStorage.getItem("authToken")
        },
        body: JSON.stringify(updates),
        method: "PATCH"
    }

    fetch(url, params)
    .then(response => {
        if (response.status < 300) {
            return response;
        }
    })
    .then(response => {
        // will get the current user and switch screens
        myUser = getCurrentUser();
        document.getElementById("loggedin-firstname").textContent = myUser.FirstName;
        document.getElementById("loggedin-lastname").textContent = myUser.LastName;
    })
    .catch(err => {
        document.getElementById("error").textContent = err;
    })
}

// Will check the local storage for a valid auth token.
// If invalid or unset, will force the user back to the landing page.
// If valid, will show the user their profile page.
function areWeAuthorized() {
    if (window.localStorage.getItem("authToken") == null) {
        document.getElementById("screen-landing").style.visibility = "visible";
        document.getElementById("screen-landing").style.display = "block";
        document.getElementById("screen-loggedin").style.visibility = "none";
        document.getElementById("screen-loggedin").style.display = "none";
        document.getElementById("loggedin-firstname").textContent = "";
        document.getElementById("loggedin-lastname").textContent = "";
        return false;
    } else {
        document.getElementById("screen-landing").style.visibility = "none";
        document.getElementById("screen-landing").style.display = "none";
        document.getElementById("screen-loggedin").style.visibility = "visible";
        document.getElementById("screen-loggedin").style.display = "block";
        return true;      
    }
    
}

function getCurrentUser() {
    if (areWeAuthorized()) {
        let url = "/v1/users/me";
        let params = {
            headers: {
                "content-type": "application/json; charset=UTF-8",
                "Authorization": window.localStorage.getItem("authToken")
            },
            method: "GET"
        }

        fetch(url, params)
        .then(response => {
            if (response.status < 300) {
                return response.json();
            }
        })
        .catch(err => {
            document.getElementById("error").textContent = err;
        })

    } else {
        return null;
    }
}