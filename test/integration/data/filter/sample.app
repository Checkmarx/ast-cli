// JavaScript application file with XSS vulnerability
function display(userInput) {
    document.getElementById("output").innerHTML = "<p>" + userInput + "</p>";
}
