document.addEventListener('DOMContentLoaded', () => {
    const notesContainer = document.getElementById('notes-container');

    const fetchBookmarks = async () => {
        try {
            const response = await fetch('/bookmarks'); // Updated endpoint
            if (!response.ok) {
                throw new Error('Failed to fetch bookmarks');
            }
            const notes = await response.json();
            renderNotes(notes);
        } catch (error) {
            console.error(error);
            notesContainer.innerHTML = '<p>Error loading bookmarks.</p>';
        }
    };

    const renderNotes = (notes) => {
        if (!notes || notes.length === 0) {
            notesContainer.innerHTML = '<p>No bookmarks yet.</p>';
            return;
        }
        notesContainer.innerHTML = '';
        notes.forEach(note => {
            const noteElement = document.createElement('div');
            noteElement.classList.add('note');
            noteElement.innerHTML = `
                <p>${note.content}</p>
                <div class="note-footer">
                    <span>${new Date(note.create_time).toLocaleString()}</span>
                    <button class="unbookmark-btn" data-note-id="${note.id}">Unbookmark</button>
                </div>
            `;
            notesContainer.appendChild(noteElement);
        });
    };

    notesContainer.addEventListener('click', async (event) => {
        if (event.target.classList.contains('unbookmark-btn')) {
            const noteId = event.target.dataset.noteId;
            try {
                const response = await fetch(`/bookmarks/${noteId}`, { // Updated endpoint
                    method: 'DELETE',
                });
                if (!response.ok) {
                    throw new Error('Failed to unbookmark note');
                }
                fetchBookmarks();
            } catch (error) {
                console.error(error);
            }
        }
    });

    fetchBookmarks();
});