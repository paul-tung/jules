import { Injectable } from '@angular/core';
import { HttpClient, HttpEvent, HttpErrorResponse, HttpParams } from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface Video {
  ID: string;
  userId: string;
  title: string;
  originalFilename: string;
  storagePath: string;
  status: string;
  uploadedAt: string; // Assuming ISO string date
  updatedAt: string; // Assuming ISO string date
  description?: string;
  tags?: string[];
  durationSeconds?: number;
  thumbnailPath?: string;
  views?: number;
}

export interface PaginatedVideosResponse {
  currentPage: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
  items: Video[];
}

export interface UpdateVideoPayload {
  title?: string;
  description?: string;
  tags?: string[];
  status?: string;
}


@Injectable({
  providedIn: 'root'
})
export class VideoService {
  private apiUrl = `${environment.apiUrl}/videos`; // Base URL for video operations

  constructor(private http: HttpClient) {}

  uploadVideo(file: File, title?: string, description?: string, tags?: string[]): Observable<HttpEvent<any>> {
    const formData: FormData = new FormData();
    formData.append('videoFile', file, file.name);

    if (title) {
      formData.append('title', title);
    }
    if (description) {
      formData.append('description', description);
    }
    if (tags && tags.length > 0) {
      tags.forEach(tag => {
        if (tag.trim() !== '') { // Ensure tag is not empty
          formData.append('tags', tag.trim());
        }
      });
    }

    return this.http.post<any>(`${this.apiUrl}/upload`, formData, {
      reportProgress: true,
      observe: 'events'
    }).pipe(
      catchError(this.handleError)
    );
  }

  assignVideoToFolder(videoId: string, folderId: string | null): Observable<Video> {
    // If folderId is empty string, treat it as null for backend to unset
    const payload = { folderId: folderId === '' ? null : folderId };
    return this.http.put<Video>(`${this.apiUrl}/${videoId}/folder`, payload).pipe(
      catchError(this.handleError)
    );
  }

  getVideos(page: number = 1, pageSize: number = 10): Observable<PaginatedVideosResponse> {
    let params = new HttpParams()
      .set('page', page.toString())
      .set('pageSize', pageSize.toString());

    return this.http.get<PaginatedVideosResponse>(this.apiUrl, { params }).pipe(
      catchError(this.handleError)
    );
  }

  getVideo(videoId: string): Observable<Video> {
    return this.http.get<Video>(`${this.apiUrl}/${videoId}`).pipe(
      catchError(this.handleError)
    );
  }

  updateVideo(videoId: string, metadata: UpdateVideoPayload): Observable<Video> {
    return this.http.put<Video>(`${this.apiUrl}/${videoId}`, metadata).pipe(
      catchError(this.handleError)
    );
  }

  deleteVideo(videoId: string): Observable<any> { // Backend returns { "message": "..." }
    return this.http.delete<any>(`${this.apiUrl}/${videoId}`).pipe(
      catchError(this.handleError)
    );
  }

  private handleError(error: HttpErrorResponse) {
    let errorMessage = 'An unknown error occurred!';
    if (error.error instanceof ErrorEvent) {
      // Client-side errors
      errorMessage = `Error: ${error.error.message}`;
    } else {
      // Server-side errors
      if (error.status === 0) {
        errorMessage = 'Could not connect to the server. Please try again later.';
      } else if (error.error && error.error.error) { // Gin Gonic error structure
        errorMessage = error.error.error;
      } else if (error.error && error.error.message) { // For delete success message, or other structures
        errorMessage = error.error.message;
      } else if (error.error && typeof error.error === 'string') {
        errorMessage = error.error;
      } else {
        errorMessage = `Server returned code: ${error.status}, error message is: ${error.message}`;
      }
    }
    return throwError(() => new Error(errorMessage));
  }
}
