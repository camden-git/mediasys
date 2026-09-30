import http, { unwrap } from './http';
import { Alias, Person, PersonImageResult } from '../types';
import { ApiResponse } from './standard';

export interface PersonImagesPage {
    items: PersonImageResult[];
    total: number;
    offset: number;
    limit: number;
    has_more: boolean;
}

export const getPeople = async (signal?: AbortSignal): Promise<Person[]> =>
    unwrap(await http.get<ApiResponse<Person[]>>('/people', { signal }));

export const getPersonById = async (id: number, signal?: AbortSignal): Promise<Person> =>
    unwrap(await http.get<ApiResponse<Person>>(`/people/${id}`, { signal }));

/** Admin variant of getPersonById that includes faces from hidden albums (requires people.manage). */
export const getPersonByIdAdmin = async (id: number, signal?: AbortSignal): Promise<Person> =>
    unwrap(await http.get<ApiResponse<Person>>(`/people/${id}/admin`, { signal }));

export const getPersonImages = async (
    personId: number,
    params: { offset?: number; limit?: number } = {},
    signal?: AbortSignal,
): Promise<PersonImagesPage> =>
    unwrap(await http.get<ApiResponse<PersonImagesPage>>(`/people/${personId}/images`, { params, signal }));

export const searchPeople = async (q: string, limit = 5, signal?: AbortSignal): Promise<Person[]> =>
    unwrap(await http.get<ApiResponse<Person[]>>('/people/search', { params: { q, limit }, signal }));

export const createPerson = async (name: string): Promise<Person> =>
    unwrap(await http.post<ApiResponse<Person>>('/people', { primary_name: name }));

export const updatePerson = async (id: number, name: string): Promise<Person> =>
    unwrap(await http.put<ApiResponse<Person>>(`/people/${id}`, { primary_name: name }));

export const deletePerson = async (id: number): Promise<void> => {
    await http.delete(`/people/${id}`);
};

export const getPersonAliases = async (personId: number, signal?: AbortSignal): Promise<Alias[]> =>
    unwrap(await http.get<ApiResponse<Alias[]>>(`/people/${personId}/aliases`, { signal }));

export const addPersonAlias = async (personId: number, name: string): Promise<Alias> =>
    unwrap(await http.post<ApiResponse<Alias>>(`/people/${personId}/aliases`, { name }));

export const deletePersonAlias = async (personId: number, aliasId: number): Promise<void> => {
    await http.delete(`/people/${personId}/aliases/${aliasId}`);
};

/** Sets (or clears) the key photo for a person. Pass null to clear. */
export const setPersonKeyPhoto = async (personId: number, faceId: number | null): Promise<Person> =>
    unwrap(await http.put<ApiResponse<Person>>(`/people/${personId}/key-photo`, { face_id: faceId ?? 0 }));
