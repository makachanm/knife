document.addEventListener('DOMContentLoaded', () => {
    const timeline = document.getElementById('timeline');

    const publicRanges = {
        0: 'Private',
        1: 'Followers Only',
        2: 'Unlisted',
        3: 'Public'
    };

    fetchNotes();

    async function fetchNotes() {
        try {
            const response = await fetch('/api/notes');
            if (!response.ok) {
                throw new Error('Could not fetch notes');
            }
            const notes = await response.json();
            renderNotes(notes);
        } catch (error) {
            timeline.innerHTML = `<p class='error-message'>Error fetching timeline: ${error.message}</p>`;
            console.error('Failed to fetch notes:', error);
        }
    }

    function renderNotes(notes) {
        if (!notes || notes.length === 0) {
            timeline.innerHTML = '<p>No notes yet. Be the first to post!</p>';
            return;
        }

        timeline.innerHTML = '';
        notes.forEach(note => {
            const noteElement = document.createElement('div');
            noteElement.className = 'note';
            noteElement.dataset.noteId = note.id;

            const createTime = new Date(note.create_time).toLocaleString();

            noteElement.innerHTML = `
                <div class='note-header'>
                    <div>
                        <span class='author'>${escapeHTML(note.author_name)}</span>
                        <span class='finger'>@${escapeHTML(note.author_finger)}</span>
                    </div>
                </div>
                ${note.cw ? `<div class='note-cw'>CW: ${escapeHTML(note.cw)}</div>` : ''}
                <div class='note-content'>${escapeHTML(note.content)}</div>
                <div class='note-meta'>
                    <a href='/notes/${note.id}' class='note-link-time'>Posted on ${createTime}</a>
                    <br />
                    <span class='public-range'>${publicRanges[note.public_range] || 'Unknown'}</span>
                </div>
                <div class='note-actions'>
                    <button class='bookmark-button' data-note-id='${note.id}'>Bookmark</button>
                    <button class='delete-button'>Delete</button>
                </div>
            `;
            timeline.appendChild(noteElement);
        });
    }

    timeline.addEventListener('click', (e) => {
        if (e.target.classList.contains('delete-button')) {
            const noteElement = e.target.closest('.note');
            const noteId = noteElement.dataset.noteId;
            if (confirm('Are you sure you want to delete this note?')) {
                deleteNote(noteId);
            }
        }
        if (e.target.classList.contains('bookmark-button')) {
            const noteElement = e.target.closest('.note');
            const noteId = noteElement.dataset.noteId;
            bookmarkNote(noteId);
        }
    });

    async function deleteNote(id) {
        try {
            const response = await fetch(`/api/notes/${id}`, { method: 'DELETE' });
            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.description || 'Failed to delete note');
            }
            fetchNotes();
        } catch (error) {
            alert(`Error deleting note: ${error.message}`);
            console.error('Failed to delete note:', error);
        }
    }

    async function bookmarkNote(id) {
        try {
            const response = await fetch(`/api/bookmarks`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ note_id: id })
            });
            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.description || 'Failed to bookmark note');
            }
            alert('Note bookmarked!');
        } catch (error) {
            alert(`Error bookmarking note: ${error.message}`);
            console.error('Failed to bookmark note:', error);
        }
    }

    function escapeHTML(str) {
        if (str === null || str === undefined) {
            return '';
        }
        return str.toString()
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }
});