export interface ApiErrorDetail {
    code: string;
    status: string;
    detail: string;
}

export interface ApiErrorResponse {
    errors?: ApiErrorDetail[];
    detail?: string;
    message?: string;
}

/**
 * Converts an error into a human readable response. Mostly just a generic helper to
 * make sure we display the message from the server back to the user if we can.
 */
export function httpErrorToHuman(error: any): string {
    if (!error) {
        return 'An unexpected error occurred.';
    }

    const responseData = error?.response?.data;
    if (responseData) {
        let data = responseData;

        // Some non-JSON requests can still return the error as a JSON block. In those cases, attempt
        // to parse it into JSON so we can display an actual error.
        if (typeof data === 'string') {
            try {
                data = JSON.parse(data);
            } catch {
                return data;
            }
        }

        if (data?.errors && Array.isArray(data.errors) && data.errors[0]?.detail) {
            return data.errors[0].detail;
        }

        if (typeof data?.detail === 'string') {
            return data.detail;
        }

        if (typeof data?.message === 'string') {
            return data.message;
        }
    }

    if (error?.message) {
        return error.message;
    }

    return 'An unexpected error occurred.';
}

export interface PaginationMetaResponse {
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

export function getPaginationSet(meta?: PaginationMetaResponse): PaginationDataSet {
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

type QueryBuilderFilterValue = string | number | boolean | null;

export interface QueryBuilderParams<FilterKeys extends string = string, SortKeys extends string = string> {
    page?: number;
    filters?: {
        [K in FilterKeys]?: QueryBuilderFilterValue | Readonly<QueryBuilderFilterValue[]>;
    };
    sorts?: {
        [K in SortKeys]?: -1 | 0 | 1 | 'asc' | 'desc' | null;
    };
}

/**
 * Helper function that parses a data object provided and builds query parameters
 * automatically. This will apply sorts and filters deterministically based on the provided values.
 */
export const withQueryBuilderParams = (
    data?: QueryBuilderParams,
): Record<string, QueryBuilderFilterValue | Readonly<QueryBuilderFilterValue[]> | string | number | undefined> => {
    if (!data) return {};

    const filters = Object.keys(data.filters || {}).reduce(
        (obj, key) => {
            const value = data.filters?.[key];

            return !value || value === '' ? obj : { ...obj, [`filter[${key}]`]: value };
        },
        {} as NonNullable<QueryBuilderParams['filters']>,
    );

    const sorts = Object.keys(data.sorts || {}).reduce((arr, key) => {
        const value = data.sorts?.[key];
        if (!value || !['asc', 'desc', 1, -1].includes(value)) {
            return arr;
        }

        return [...arr, (value === -1 || value === 'desc' ? '-' : '') + key];
    }, [] as string[]);

    return {
        ...filters,
        sort: !sorts.length ? undefined : sorts.join(','),
        page: data.page,
    };
};



