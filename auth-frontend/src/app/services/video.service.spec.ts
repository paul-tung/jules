import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController, TestRequest } from '@angular/common/http/testing';
import { HttpEventType, HttpResponse } from '@angular/common/http';
import { VideoService, PaginatedVideosResponse, Video, UpdateVideoPayload } from './video'; // Use correct path
import { environment } from '../../environments/environment';

describe('VideoService', () => {
  let service: VideoService;
  let httpMock: HttpTestingController;
  const mockApiUrl = `${environment.apiUrl}/videos`;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [VideoService]
    });
    service = TestBed.inject(VideoService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify(); // Ensure no outstanding requests.
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('uploadVideo', () => {
    it('should upload a file and report progress', (done) => {
      const mockFile = new File(['dummy content'], 'video.mp4', { type: 'video/mp4' });
      const mockTitle = 'Test Video';
      const mockDescription = 'A test video description';
      const mockTags = ['test', 'video'];

      const mockUploadEvents = [
        { type: HttpEventType.Sent }, // Optional: test for Sent event if your UI cares
        { type: HttpEventType.UploadProgress, loaded: 10, total: 100 },
        { type: HttpEventType.UploadProgress, loaded: 50, total: 100 },
        new HttpResponse({ body: { id: '123', message: 'Uploaded!' }, status: 201 }) // Backend response on completion
      ];
      let eventIndex = 0;

      service.uploadVideo(mockFile, mockTitle, mockDescription, mockTags).subscribe(
        event => {
          expect(event.type).toEqual(mockUploadEvents[eventIndex].type);
          if (event.type === HttpEventType.UploadProgress) {
            expect((event as any).loaded).toEqual((mockUploadEvents[eventIndex] as any).loaded);
            expect((event as any).total).toEqual((mockUploadEvents[eventIndex] as any).total);
          } else if (event instanceof HttpResponse) {
            expect(event.status).toBe(201);
            expect(event.body).toEqual({ id: '123', message: 'Uploaded!' });
            done();
          }
          eventIndex++;
        },
        () => fail('Upload should succeed')
      );

      const req = httpMock.expectOne(`${mockApiUrl}/upload`);
      expect(req.request.method).toBe('POST');
      expect(req.request.reportProgress).toBeTrue();

      // Check FormData content
      const formData = req.request.body as FormData;
      expect(formData.get('videoFile') instanceof File).toBeTrue();
      expect((formData.get('videoFile') as File).name).toBe('video.mp4');
      expect(formData.get('title')).toBe(mockTitle);
      expect(formData.get('description')).toBe(mockDescription);
      expect(formData.getAll('tags')).toEqual(mockTags);


      // Simulate backend sending events
      req.event(mockUploadEvents[0]); // Sent
      req.event(mockUploadEvents[1]); // Progress 1
      req.event(mockUploadEvents[2]); // Progress 2
      req.flush(mockUploadEvents[3].body, { status: mockUploadEvents[3].status, statusText: 'Created' }); // Final response
    });

    it('should handle upload error', (done) => {
        const mockFile = new File(['dummy content'], 'video.mp4', { type: 'video/mp4' });
        service.uploadVideo(mockFile).subscribe({
            next: () => fail('should have failed with an error'),
            error: (error) => {
                expect(error.message).toContain('Upload failed');
                done();
            }
        });

        const req = httpMock.expectOne(`${mockApiUrl}/upload`);
        req.flush({ error: 'Upload failed' }, { status: 500, statusText: 'Server Error' });
    });
  });

  describe('getVideos', () => {
    it('should fetch videos with pagination parameters', (done) => {
      const mockResponse: PaginatedVideosResponse = {
        currentPage: 1, pageSize: 10, totalItems: 1, totalPages: 1,
        items: [{ ID: '1', title: 'Test Video' } as Video]
      };

      service.getVideos(1, 10).subscribe(response => {
        expect(response).toEqual(mockResponse);
        done();
      });

      const req = httpMock.expectOne(`${mockApiUrl}?page=1&pageSize=10`);
      expect(req.request.method).toBe('GET');
      req.flush(mockResponse);
    });
  });

  describe('getVideo', () => {
    it('should fetch a single video by ID', (done) => {
      const mockVideo: Video = { ID: '1', title: 'Test Video Details' } as Video;
      const videoId = '1';

      service.getVideo(videoId).subscribe(video => {
        expect(video).toEqual(mockVideo);
        done();
      });

      const req = httpMock.expectOne(`${mockApiUrl}/${videoId}`);
      expect(req.request.method).toBe('GET');
      req.flush(mockVideo);
    });
  });

  describe('updateVideo', () => {
    it('should send a PUT request to update video metadata', (done) => {
      const videoId = '1';
      const payload: UpdateVideoPayload = { title: 'Updated Title' };
      const mockUpdatedVideo: Video = { ID: videoId, title: 'Updated Title' } as Video;

      service.updateVideo(videoId, payload).subscribe(video => {
        expect(video).toEqual(mockUpdatedVideo);
        done();
      });

      const req = httpMock.expectOne(`${mockApiUrl}/${videoId}`);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual(payload);
      req.flush(mockUpdatedVideo);
    });
  });

  describe('deleteVideo', () => {
    it('should send a DELETE request to remove a video', (done) => {
      const videoId = '1';
      const mockResponse = { message: 'Video deleted successfully' };

      service.deleteVideo(videoId).subscribe(response => {
        expect(response).toEqual(mockResponse);
        done();
      });

      const req = httpMock.expectOne(`${mockApiUrl}/${videoId}`);
      expect(req.request.method).toBe('DELETE');
      req.flush(mockResponse);
    });
  });

  describe('assignVideoToFolder', () => {
    it('should send a PUT request to assign video to a folder', (done) => {
      const videoId = 'vid1';
      const folderId = 'folder1';
      const mockUpdatedVideo: Video = { ID: videoId, folderId: folderId } as Video;

      service.assignVideoToFolder(videoId, folderId).subscribe(video => {
        expect(video).toEqual(mockUpdatedVideo);
        done();
      });

      const req = httpMock.expectOne(`${mockApiUrl}/${videoId}/folder`);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual({ folderId: folderId });
      req.flush(mockUpdatedVideo);
    });

    it('should send a PUT request to unassign video from a folder (folderId is null)', (done) => {
      const videoId = 'vid1';
      const mockUpdatedVideo: Video = { ID: videoId, folderId: null } as Video;

      service.assignVideoToFolder(videoId, null).subscribe(video => {
        expect(video).toEqual(mockUpdatedVideo);
        done();
      });

      const req = httpMock.expectOne(`${mockApiUrl}/${videoId}/folder`);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual({ folderId: null });
      req.flush(mockUpdatedVideo);
    });
  });

});
