import { PaginatedResult, PaginationRequest } from '../standard';

// The backend caps per_page at 100, so "everything" means walking the pages.
const MAX_PER_PAGE = 100;

export const fetchAllPages = async <T>(
    fetchPage: (params: PaginationRequest) => Promise<PaginatedResult<T>>,
): Promise<T[]> => {
    const first = await fetchPage({ page: 1, perPage: MAX_PER_PAGE });
    const items = [...first.items];
    const totalPages = first.pagination.totalPages;
    for (let page = 2; page <= totalPages; page++) {
        const next = await fetchPage({ page, perPage: MAX_PER_PAGE });
        items.push(...next.items);
    }
    return items;
};
