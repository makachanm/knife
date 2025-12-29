// note-renderer.js

const publicRanges = {
    0: 'Private',
    1: 'Followers Only',
    2: 'Unlisted',
    3: 'Public'
};

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

function toggleCW(button) {
    const container = button.closest('.cw-container');
    const content = container.querySelector('.cw-content');
    if (content.classList.contains('hidden')) {
        content.classList.remove('hidden');
        button.textContent = 'Hide';
    } else {
        content.classList.add('hidden');
        button.textContent = 'Show';
    }
}

// Make toggleCW global so it works with inline onclick handlers
window.toggleCW = toggleCW;

function createNoteElement(note) {
    const noteElement = document.createElement('div');
    noteElement.className = 'note';
    noteElement.dataset.noteId = note.id;

    const createTime = new Date(note.create_time).toLocaleString();

    let contentHTML = '';
    if (note.cw) {
        contentHTML = `
            <div class="cw-container">
                <div class="cw-header">
                    <span class="cw-text">${escapeHTML(note.cw)}</span>
                    <button class="cw-toggle-button" onclick="toggleCW(this)">Show</button>
                </div>
                <div class="cw-content hidden">
                    <div class="note-content-inner"></div>
                </div>
            </div>
        `;
    } else {
        contentHTML = `<div class="note-content-inner"></div>`;
    }

    noteElement.innerHTML = `
        <div class='note-header'>
            <div>
                <span class='author'>${escapeHTML(note.author_name)}</span>
                <span class='finger'>@${escapeHTML(note.author_finger)}</span>
            </div>
        </div>
        <hr />
        <div class='note-content'>
            ${contentHTML}
        </div>
        <div class='note-meta'>
            <a href='/notes/${note.id}' class='note-link-time'>Posted on ${createTime}</a>
            ${note.category ? `<span> | Category: <a href="/category/${encodeURIComponent(note.category)}">${escapeHTML(note.category)}</a></span>` : ''}
            <br />
            <span class='public-range'>${publicRanges[note.public_range] || 'Unknown'}</span>
            <br />
            <span>Host: ${escapeHTML(note.host)}</span>
            ${note.likes > 0 ? `<span> | ${note.likes} Likes</span>` : ''}
            ${note.shares > 0 ? `<span> | ${note.shares} Shares</span>` : ''}
        </div>
        <div class='note-actions'>
             <!-- Actions can be customized or hidden via CSS if needed, but included for consistency -->
             <!-- Only show actions if needed, or maybe add them dynamically outside -->
        </div>
    `;

    // Insert HTML content into the note-content-inner div safely
    // Assuming note.content is safe HTML (sanitized on backend or trusted)
    const noteContentInner = noteElement.querySelector('.note-content-inner');
    if (noteContentInner) {
        noteContentInner.innerHTML = note.content;
    }

    return noteElement;
}

// Export functions if using modules, but for simple script tags:
window.NoteRenderer = {
    createNoteElement,
    escapeHTML
};
