// Greets the visitor by the time of day.
const hour = new Date().getHours();
const greeting = hour < 12 ? "Good morning" : "Good afternoon";
document.querySelector("p").prepend(`${greeting}! `);
