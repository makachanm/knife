document.addEventListener('DOMContentLoaded', () => {
    const categoriesList = document.getElementById('categories-list');

    fetchCategories();

    async function fetchCategories() {
        try {
            const response = await fetch('/api/category');
            if (!response.ok) {
                throw new Error('Could not fetch categories');
            }
            const categories = await response.json();
            renderCategories(categories);
        } catch (error) {
            categoriesList.innerHTML = `<p class='error-message'>Error fetching categories: ${error.message}</p>`;
            console.error('Failed to fetch categories:', error);
        }
    }

    function renderCategories(categories) {
        if (!categories || categories.length === 0) {
            categoriesList.innerHTML = '<p>No categories found.</p>';
            return;
        }

        categoriesList.innerHTML = '';
        const list = document.createElement('ul');
        list.className = 'category-list-items';

        categories.forEach(category => {
            const item = document.createElement('li');
            const link = document.createElement('a');
            link.href = `/category/${encodeURIComponent(category)}`;
            link.textContent = category;
            link.className = 'category-pill'; // We'll add some styling for this
            item.appendChild(link);
            list.appendChild(item);
        });

        categoriesList.appendChild(list);
    }
});
