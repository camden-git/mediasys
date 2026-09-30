interface PaginationMetaResponse {
    total: number;
    count: number;
    per_page: number;
    current_page: number;
    total_pages: number;
}

export interface PaginationDataSet {
    total: number;
    count: number;
    perPage: number;
    currentPage: number;
    totalPages: number;
}

export interface PaginatedResult<T> {
    items: T[];
    pagination: PaginationDataSet;
}

export interface ApiResponse<T> {
    data: T;
    meta?: {
        pagination?: PaginationMetaResponse;
    };
}

export interface PaginationRequest {
    page?: number;
    perPage?: number;
}

export function toPaginationQuery(params?: PaginationRequest): Record<string, number | undefined> {
    if (!params) {
        return {};
    }

    return {
        page: params.page,
        per_page: params.perPage,
    };
}

function getPaginationSet(meta?: PaginationMetaResponse): PaginationDataSet {
    if (!meta) {
        return {
            total: 0,
            count: 0,
            perPage: 0,
            currentPage: 0,
            totalPages: 0,
        };
    }

    return {
        total: meta.total,
        count: meta.count,
        perPage: meta.per_page,
        currentPage: meta.current_page,
        totalPages: meta.total_pages,
    };
}

export function toPaginatedResult<T>(response: ApiResponse<T[]>): PaginatedResult<T> {
    return {
        items: response.data,
        pagination: getPaginationSet(response.meta?.pagination),
    };
}
