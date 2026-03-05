// api.js — HTTP wrappers for the backend REST API.
// All functions return data; none touch the DOM or hold state.

export async function fetchBriefing() {
    const resp = await fetch('/api/briefing');
    return resp.json();
}

export async function fetchTodos() {
    const resp = await fetch('/api/todos');
    return resp.json();
}

export async function createTodo(title) {
    await fetch('/api/todos', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title, priority: 1, status: 0 }),
    });
    return fetchTodos();
}

export async function toggleTodo(id, currentStatus) {
    const newStatus = currentStatus === TODO_STATUS.DONE ? TODO_STATUS.PENDING : TODO_STATUS.DONE;
    await fetch(`/api/todos/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: newStatus }),
    });
    return fetchTodos();
}

export async function deleteTodo(id) {
    await fetch(`/api/todos/${id}`, { method: 'DELETE' });
    return fetchTodos();
}

// ---- Domain constants (mirror models/models.go) ----

export const TODO_STATUS = {
    PENDING:     0,
    IN_PROGRESS: 1,
    DONE:        2,
    ARCHIVED:    3,
};

export const PRIORITY = {
    LOW:    0,
    MEDIUM: 1,
    HIGH:   2,
    URGENT: 3,
};
