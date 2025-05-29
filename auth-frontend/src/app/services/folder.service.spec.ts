import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { FolderService, Folder } from './folder'; // Use correct path
import { environment } from '../../environments/environment';

describe('FolderService', () => {
  let service: FolderService;
  let httpMock: HttpTestingController;
  const mockApiUrl = `${environment.apiUrl}/folders`;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [FolderService]
    });
    service = TestBed.inject(FolderService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify(); // Ensure no outstanding requests.
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('createFolder', () => {
    it('should create a root folder and return it', (done) => {
      const folderName = 'New Root Folder';
      const mockResponse: Folder = { ID: '1', name: folderName, userId: 'user1', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() };

      service.createFolder(folderName).subscribe(folder => {
        expect(folder).toEqual(mockResponse);
        done();
      });

      const req = httpMock.expectOne(mockApiUrl);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({ name: folderName });
      req.flush(mockResponse);
    });

    it('should create a sub-folder with parentFolderId and return it', (done) => {
      const folderName = 'New Sub Folder';
      const parentId = 'parent1';
      const mockResponse: Folder = { ID: '2', name: folderName, parentFolderId: parentId, userId: 'user1', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() };

      service.createFolder(folderName, parentId).subscribe(folder => {
        expect(folder).toEqual(mockResponse);
        done();
      });

      const req = httpMock.expectOne(mockApiUrl);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({ name: folderName, parentFolderId: parentId });
      req.flush(mockResponse);
    });
  });

  describe('getFolders', () => {
    it('should fetch root folders when no parentFolderId is provided (or "root")', (done) => {
      const mockResponse: Folder[] = [
        { ID: '1', name: 'Root Folder 1', userId: 'user1', createdAt: '', updatedAt: '' },
        { ID: '2', name: 'Root Folder 2', userId: 'user1', createdAt: '', updatedAt: '' }
      ];

      service.getFolders('root').subscribe(folders => { // Testing with 'root' explicitly
        expect(folders.length).toBe(2);
        expect(folders).toEqual(mockResponse);
        done();
      });

      const req = httpMock.expectOne(`${mockApiUrl}?parentFolderId=root`);
      expect(req.request.method).toBe('GET');
      req.flush(mockResponse);
    });
    
    it('should fetch root folders when parentFolderId is undefined (service defaults to "root")', (done) => {
      const mockResponse: Folder[] = [ { ID: '1', name: 'Root Folder 1', userId: 'user1', createdAt: '', updatedAt: '' } ];
      service.getFolders().subscribe(folders => {
        expect(folders).toEqual(mockResponse);
        done();
      });
      const req = httpMock.expectOne(`${mockApiUrl}?parentFolderId=root`); // Service defaults to 'root'
      expect(req.request.method).toBe('GET');
      req.flush(mockResponse);
    });


    it('should fetch sub-folders for a given parentFolderId', (done) => {
      const parentId = 'parent1';
      const mockResponse: Folder[] = [
        { ID: 'sub1', name: 'Sub Folder 1', parentFolderId: parentId, userId: 'user1', createdAt: '', updatedAt: '' }
      ];

      service.getFolders(parentId).subscribe(folders => {
        expect(folders.length).toBe(1);
        expect(folders).toEqual(mockResponse);
        done();
      });

      const req = httpMock.expectOne(`${mockApiUrl}?parentFolderId=${parentId}`);
      expect(req.request.method).toBe('GET');
      req.flush(mockResponse);
    });
  });

  describe('updateFolder', () => {
    it('should send a PUT request to update a folder name', (done) => {
      const folderId = '1';
      const newName = 'Updated Folder Name';
      const mockResponse: Folder = { ID: folderId, name: newName, userId: 'user1', createdAt: '', updatedAt: '' };

      service.updateFolder(folderId, newName).subscribe(folder => {
        expect(folder).toEqual(mockResponse);
        done();
      });

      const req = httpMock.expectOne(`${mockApiUrl}/${folderId}`);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual({ name: newName });
      req.flush(mockResponse);
    });
  });

  describe('deleteFolder', () => {
    it('should send a DELETE request to remove a folder', (done) => {
      const folderId = '1';
      const mockResponse = { message: 'Folder deleted successfully' };

      service.deleteFolder(folderId).subscribe(response => {
        expect(response).toEqual(mockResponse);
        done();
      });

      const req = httpMock.expectOne(`${mockApiUrl}/${folderId}`);
      expect(req.request.method).toBe('DELETE');
      req.flush(mockResponse);
    });
  });

  describe('handleError', () => {
    it('should handle client-side error', (done) => {
        service.getFolders().subscribe({
            next: () => fail('should have failed'),
            error: (error: Error) => {
                expect(error.message).toContain('Client-side error');
                done();
            }
        });
        const req = httpMock.expectOne(`${mockApiUrl}?parentFolderId=root`);
        req.error(new ErrorEvent('Network error', { message: 'Client down' }));
    });

    it('should handle server-side error (Gin Gonic like)', (done) => {
        service.getFolders().subscribe({
            next: () => fail('should have failed'),
            error: (error: Error) => {
                expect(error.message).toBe('Server specific error message');
                done();
            }
        });
        const req = httpMock.expectOne(`${mockApiUrl}?parentFolderId=root`);
        req.flush({ error: 'Server specific error message' }, { status: 500, statusText: 'Server Error' });
    });

    it('should handle server-side error (message field)', (done) => {
        service.deleteFolder('1').subscribe({ // delete often returns {message: ...} on success too
            next: () => fail('should have failed'),
            error: (error: Error) => {
                expect(error.message).toBe('Deletion failed due to conflict');
                done();
            }
        });
        const req = httpMock.expectOne(`${mockApiUrl}/1`);
        req.flush({ message: 'Deletion failed due to conflict' }, { status: 409, statusText: 'Conflict' });
    });
    
    it('should handle server-side error (plain string)', (done) => {
        service.getFolders().subscribe({
            next: () => fail('should have failed'),
            error: (error: Error) => {
                expect(error.message).toBe('Plain text error from server');
                done();
            }
        });
        const req = httpMock.expectOne(`${mockApiUrl}?parentFolderId=root`);
        req.flush('Plain text error from server', { status: 403, statusText: 'Forbidden' });
    });
  });

});
