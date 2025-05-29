import { Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse, HttpParams } from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface Folder {
  ID: string;
  userId: string;
  name: string;
  parentFolderId?: string | null; // Nullable for root folders
  createdAt: string; // Assuming ISO string date
  updatedAt: string; // Assuming ISO string date
}

@Injectable({
  providedIn: 'root'
})
export class FolderService {
  private apiUrl = `${environment.apiUrl}/folders`;

  constructor(private http: HttpClient) {}

  createFolder(name: string, parentFolderId?: string): Observable<Folder> {
    const payload: { name: string; parentFolderId?: string } = { name };
    if (parentFolderId) {
      payload.parentFolderId = parentFolderId;
    }
    return this.http.post<Folder>(this.apiUrl, payload).pipe(
      catchError(this.handleError)
    );
  }

  // parentFolderId can be actual ID, "root", or undefined (for all folders, though backend defaults to root)
  getFolders(parentFolderId?: string): Observable<Folder[]> {
    let params = new HttpParams();
    if (parentFolderId) {
      params = params.set('parentFolderId', parentFolderId);
    } else {
      // Consistent with backend: if not specified, list root folders
      params = params.set('parentFolderId', 'root');
    }
    return this.http.get<Folder[]>(this.apiUrl, { params }).pipe(
      catchError(this.handleError)
    );
  }

  updateFolder(folderId: string, name: string): Observable<Folder> {
    return this.http.put<Folder>(`${this.apiUrl}/${folderId}`, { name }).pipe(
      catchError(this.handleError)
    );
  }

  deleteFolder(folderId: string): Observable<any> { // Backend returns { "message": "..." }
    return this.http.delete<any>(`${this.apiUrl}/${folderId}`).pipe(
      catchError(this.handleError)
    );
  }

  private handleError(error: HttpErrorResponse) {
    let errorMessage = 'An unknown error occurred in FolderService!';
    if (error.error instanceof ErrorEvent) {
      errorMessage = `Client-side error: ${error.error.message}`;
    } else {
      if (error.status === 0) {
        errorMessage = 'Could not connect to the server. Please try again later.';
      } else if (error.error && error.error.error) { // Gin Gonic error structure
        errorMessage = error.error.error;
      } else if (error.error && error.error.message) { // For delete success message
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
