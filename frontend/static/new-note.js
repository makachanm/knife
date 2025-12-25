document.addEventListener("DOMContentLoaded", () => {
    const contentField = document.getElementById("content");
    const cwField = document.getElementById("cw");
    const categoryField = document.getElementById("category");
    const visibilityField = document.getElementById("public_range");
    const saveDraftButton = document.getElementById("save-draft");
    const formError = document.getElementById("form-error");

    let currentDraftId = null;

    // Load draft from the server
    const loadDraft = async () => {
        try {
            const response = await fetch("/api/drafts");
            if (response.ok) {
                const drafts = await response.json();
                if (drafts.length > 0) {
                    const latestDraft = drafts[0]; // Load the most recent draft
                    currentDraftId = latestDraft.id;
                    contentField.value = latestDraft.content || "";
                    cwField.value = latestDraft.cw || "";
                    if (latestDraft.category) categoryField.value = latestDraft.category;
                    visibilityField.value = latestDraft.public_range || "3";
                }
            }
        } catch (err) {
            console.error("Failed to load drafts:", err);
        }
    };

    // Save draft to the server
    const saveDraft = async () => {
        const draftData = {
            content: contentField.value,
            cw: cwField.value,
            category: categoryField.value,
            public_range: visibilityField.value,
        };

        try {
            const response = currentDraftId
                ? await fetch(`/api/drafts/${currentDraftId}`, {
                      method: "PUT",
                      headers: {
                          "Content-Type": "application/json",
                      },
                      body: JSON.stringify(draftData),
                  })
                : await fetch("/api/drafts", {
                      method: "POST",
                      headers: {
                          "Content-Type": "application/json",
                      },
                      body: JSON.stringify(draftData),
                  });

            if (response.ok) {
                const draft = await response.json();
                currentDraftId = draft.id;
                formError.textContent = "Draft saved successfully.";
                formError.style.color = "green";
            } else {
                const error = await response.json();
                formError.textContent = error.message || "Failed to save draft.";
                formError.style.color = "red";
            }
        } catch (err) {
            formError.textContent = "An error occurred while saving the draft.";
            formError.style.color = "red";
            console.error("Failed to save draft:", err);
        }
    };

    // Clear draft from the server after posting
    const clearDraft = async () => {
        if (currentDraftId) {
            try {
                await fetch(`/api/drafts/${currentDraftId}`, {
                    method: "DELETE",
                });
                currentDraftId = null;
            } catch (err) {
                console.error("Failed to delete draft:", err);
            }
        }
    };

    // Handle form submission
    const noteForm = document.getElementById("note-form");
    noteForm.addEventListener("submit", async (event) => {
        event.preventDefault();

        const noteData = {
            content: contentField.value,
            cw: cwField.value,
            category: categoryField.value,
            public_range: visibilityField.value,
        };

        try {
            const response = await fetch("/api/notes", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(noteData),
            });

            if (response.ok) {
                await clearDraft();
                formError.textContent = "Note posted successfully!";
                formError.style.color = "green";
                noteForm.reset();
            } else {
                const error = await response.json();
                formError.textContent = error.message || "Failed to post note.";
                formError.style.color = "red";
            }
        } catch (err) {
            formError.textContent = "An error occurred while posting the note.";
            formError.style.color = "red";
            console.error("Failed to post note:", err);
        }
    });

    // Attach event listener to save draft button
    saveDraftButton.addEventListener("click", saveDraft);

    // Load draft on page load
    loadDraft();
});
