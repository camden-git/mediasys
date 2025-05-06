export interface Album {
    id: number;
    name: string;
    slug: string;
    description?: string;
    folder_path: string;
    banner_image_path?: string;
    zip_path?: string;
    zip_size?: number;
    zip_status: string;
    zip_last_generated_at?: number;
    zip_last_requested_at?: number;

    created_at: number;
    updated_at: number;
}

export interface FileInfo {
    name: string;
    path: string;
    is_dir: boolean;
    size: number;
    mod_time: number;
    thumbnail_path?: string;
    width?: number;
    height?: number;

    aperture?: number;
    shutter_speed?: string;
    iso?: number;
    focal_length?: number;
    lens_make?: string;
    lens_model?: string;
    camera_make?: string;
    camera_model?: string;
    taken_at?: number;

    thumbnail_status?: string;
    metadata_status?: string;
    detection_status?: string;
}

export interface DirectoryListing {
    path: string;
    files: FileInfo[];
    parent?: string;
}
