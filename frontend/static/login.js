document.addEventListener("DOMContentLoaded", () => {
    const loginForm = document.getElementById("login-form");
    const logoutButton = document.getElementById("logout-button");

    // Check login status
    fetch("/api/auth/status", {
        method: "GET",
        credentials: "include", // Include cookies in the request
    })
        .then((response) => response.json())
        .then((data) => {
            if (data.logged_in) {
                loginForm.style.display = "none";
                logoutButton.style.display = "block";
            } else {
                loginForm.style.display = "block";
                logoutButton.style.display = "none";
            }
        });

    // Handle login
    loginForm.addEventListener("submit", (e) => {
        e.preventDefault();
        const password = document.getElementById("password").value;

        fetch("/api/login", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ password }),
            credentials: "include",
        })
            .then((response) => {
                if (!response.ok) throw new Error("Login failed");
                return response.json();
            })
            .then(() => {
                alert("Login successful!");
                window.location.href = "/";
            })
            .catch((error) => alert(error.message));
    });

    // Handle logout
    logoutButton.addEventListener("click", () => {
        fetch("/api/logout", {
            method: "POST",
            credentials: "include",
        })
            .then((response) => {
                if (!response.ok) throw new Error("Logout failed");
                return response.json();
            })
            .then(() => {
                alert("Logout successful!");
                window.location.reload();
            })
            .catch((error) => alert(error.message));
    });
});