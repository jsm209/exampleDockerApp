function getData () {
    let link = "https://api.infoclass.me/v1/summary?url=" + document.getElementById("link").value;
    fetch(link, {
            mode: "cors"
        })
        .then(function(response) {
            return JSON.parse(JSON.stringify(response))
        })
        .then(function(response) {
            document.getElementById("errorFeedback").innerText = "No errors!";
            
            let title = document.createElement("p");
            title.innerText = "Title: " + response.title;
            document.getElementById("content").append(title);

            let description = document.createElement("p");
            description.innerText = "Description: " + response.description;
            document.getElementById("content").append(description);

            for (let i = 0; i < response.images; i++) {
                let image = document.createElement("img");
                image.setAttribute("src", response.images[i].URL);
                document.getElementById("content").append(image);
            }
        })
        .catch(function(err) {
            document.getElementById("errorFeedback").innerText = err;
        }); 
}