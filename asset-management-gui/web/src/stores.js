import { writable } from "svelte/store";

export const containers = writable([]);
export const online = writable(null);
export const query = writable("");
export const statusFilter = writable("");
export const typeFilter = writable("");
export const sort = writable("id");