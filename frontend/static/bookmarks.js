document.addEventListener('DOMContentLoaded', () => {
    const notesContainer = document.getElementById('notes-container');

    // Fetch bookmarks from the API
    fetch('/api/bookmarks')
        .then((response) => {
            if (!response.ok) {
                throw new Error('Failed to fetch bookmarks');
            }
            return response.json();
        })
        .then((notes) => {
            if (notes.length === 0) {
                notesContainer.innerHTML = '<p>No bookmarks found.</p>';
                return;
            }

            notes.forEach((note) => {
                const noteElement = document.createElement('div');
                noteElement.className = 'note';
                noteElement.innerHTML = `
                    <p>${note.content}</p>
                    <p class="note-meta">By ${note.author} on ${new Date(note.created).toLocaleDateString()}</p>
                `;
                notesContainer.appendChild(noteElement);
            });
        })
        .catch((error) => {
            notesContainer.innerHTML = `<p class="error-message">${error.message}</p>`;
        });
});