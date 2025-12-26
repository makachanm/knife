document.addEventListener('DOMContentLoaded', () => {
    const notesContainer = document.getElementById('notes-container');

    fetchBookmarks();

    async function fetchBookmarks() {
        try {
            const response = await fetch('/api/bookmarks');
            if (!response.ok) {
                throw new Error('Failed to fetch bookmarks');
            }
            const notes = await response.json();
            renderBookmarks(notes);
        } catch (error) {
            notesContainer.innerHTML = `<p class="error-message">${error.message}</p>`;
        }
    }

    function renderBookmarks(notes) {
        if (!notes || notes.length === 0) {
            notesContainer.innerHTML = '<p>No bookmarks found.</p>';
            return;
        }

        notesContainer.innerHTML = '';
        notes.forEach((note) => {
            const noteElement = NoteRenderer.createNoteElement(note);
            
            // Add "Remove Bookmark" button
            const actionsDiv = noteElement.querySelector('.note-actions');
            if (actionsDiv) {
                const removeBtn = document.createElement('button');
                removeBtn.textContent = 'Remove Bookmark';
                removeBtn.className = 'delete-button'; // Re-using delete style for now, or could use bookmark style
                // Or create a new class .remove-bookmark-button
                removeBtn.onclick = () => removeBookmark(note.id);
                actionsDiv.appendChild(removeBtn);
            }

            notesContainer.appendChild(noteElement);
        });
    }

    async function removeBookmark(noteId) {
        if (!confirm('Are you sure you want to remove this bookmark?')) {
            return;
        }

        try {
            const response = await fetch(`/api/bookmarks/${noteId}`, {
                method: 'DELETE'
            });
            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.description || 'Failed to remove bookmark');
            }
            // Refresh list
            fetchBookmarks();
        } catch (error) {
            alert(`Error removing bookmark: ${error.message}`);
        }
    }
});
