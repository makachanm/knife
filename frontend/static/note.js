document.addEventListener('DOMContentLoaded', () => {
    const noteContainer = document.getElementById('note-container');
    const pathParts = window.location.pathname.split('/');
    const noteId = pathParts[pathParts.length - 1];

    const publicRanges = {
        0: 'Private',
        1: 'Followers Only',
        2: 'Unlisted',
        3: 'Public'
    };

    if (!noteId) {
        noteContainer.innerHTML = '<p class="error-message">Could not determine note ID from URL.</p>';
        return;
    }

    fetchNote();

    function fetchNote() {
        fetch(`/api/notes/${noteId}`)
            .then(response => {
                if (!response.ok) {
                    if (response.status === 404) {
                        throw new Error('Note not found.');
                    }
                    throw new Error('Failed to fetch note. Status: ' + response.status);
                }
                return response.json();
            })
            .then(note => {
                renderNote(note);
            })
            .catch(error => {
                noteContainer.innerHTML = `<p class="error-message">${error.message}</p>`;
            });
    }

    function renderNote(note) {
        const createTime = new Date(note.create_time).toLocaleString();

        noteContainer.innerHTML = `
            <div class="note" data-note-id="${note.id}">
                <div class='note-header'>
                    <div>
                        <span class='author'>${escapeHTML(note.author_name)}</span>
                        <span class='finger'>@${escapeHTML(note.author_finger)}</span>
                    </div>
                </div>
                ${note.cw ? `<div class='note-cw'>CW: ${escapeHTML(note.cw)}</div>` : ''}
                <div class='note-content'><iframe>${note.content}</iframe></div>
                <div class='note-meta'>
                    <span>Posted on ${createTime}</span>
                    <br />
                    <span class='public-range'>${publicRanges[note.public_range] || 'Unknown'}</span>
                    <br />
                    <span>Host: ${escapeHTML(note.host)}</span>
                    <br />
                    <span>URI: <a href="${escapeHTML(note.uri)}">${escapeHTML(note.uri)}</a></span>
                </div>
                <div class='note-actions'>
                    <button class='bookmark-button' data-note-id='${note.id}'>Bookmark</button>
                    <button class='delete-button'>Delete</button>
                </div>
            </div>
        `;
    }

    noteContainer.addEventListener('click', (e) => {
        if (e.target.classList.contains('delete-button')) {
            if (confirm('Are you sure you want to delete this note?')) {
                deleteNote(noteId);
            }
        }
        if (e.target.classList.contains('bookmark-button')) {
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
            // On successful deletion, redirect to the homepage
            window.location.href = '/';
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
                body: JSON.stringify({ note_id: parseInt(id, 10) })
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
