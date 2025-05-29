import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { HttpClientTestingModule } from '@angular/common/http/testing'; // For VideoService if not fully mocked
import { HttpEventType, HttpResponse, HttpProgressEvent, HttpSentEvent } from '@angular/common/http';
import { of, throwError, Subject } from 'rxjs';
import { By } from '@angular/platform-browser';

import { VideoUploadComponent } from './video-upload';
import { VideoService } from '../../services/video';

describe('VideoUploadComponent', () => {
  let component: VideoUploadComponent;
  let fixture: ComponentFixture<VideoUploadComponent>;
  let mockVideoService: jasmine.SpyObj<VideoService>;

  beforeEach(async () => {
    mockVideoService = jasmine.createSpyObj('VideoService', ['uploadVideo']);

    await TestBed.configureTestingModule({
      imports: [
        CommonModule,
        FormsModule,
        HttpClientTestingModule, // VideoService is providedIn: 'root' and might be injected
        VideoUploadComponent // Standalone component
      ],
      providers: [
        { provide: VideoService, useValue: mockVideoService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(VideoUploadComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize with default values', () => {
    expect(component.selectedFile).toBeNull();
    expect(component.videoTitle).toBe('');
    expect(component.videoDescription).toBe('');
    expect(component.videoTags).toBe('');
    expect(component.uploadProgress).toBe(0);
    expect(component.successMessage).toBe('');
    expect(component.errorMessage).toBe('');
    expect(component.uploadInProgress).toBeFalse();
  });

  describe('onFileSelected', () => {
    it('should update selectedFile when a file is chosen', () => {
      const mockFile = new File(['dummy'], 'test.mp4', { type: 'video/mp4' });
      const mockEvent = {
        currentTarget: { files: [mockFile] }
      } as unknown as Event; // Cast to Event after constructing

      component.onFileSelected(mockEvent);

      expect(component.selectedFile).toEqual(mockFile);
      expect(component.successMessage).toBe('');
      expect(component.errorMessage).toBe('');
      expect(component.uploadProgress).toBe(0);
    });

    it('should set selectedFile to null if no file is chosen', () => {
      const mockEvent = {
        currentTarget: { files: [] } // Empty file list
      } as unknown as Event;

      component.selectedFile = new File(['dummy'], 'test.mp4', { type: 'video/mp4' }); // Pre-select a file
      component.onFileSelected(mockEvent);
      expect(component.selectedFile).toBeNull();
    });
  });

  describe('onUpload', () => {
    it('should show error if no file is selected', () => {
      component.selectedFile = null;
      component.onUpload();
      expect(component.errorMessage).toBe('Please select a video file to upload.');
      expect(mockVideoService.uploadVideo).not.toHaveBeenCalled();
    });

    it('should call VideoService.uploadVideo with correct data and handle progress events', fakeAsync(() => {
      const mockFile = new File(['content'], 'video.mp4', { type: 'video/mp4' });
      component.selectedFile = mockFile;
      component.videoTitle = 'Test Title';
      component.videoDescription = 'Test Desc';
      component.videoTags = 'tag1, tag2';

      // Use a Subject to manually emit events for the observable
      const uploadEvents = new Subject<any>();
      mockVideoService.uploadVideo.and.returnValue(uploadEvents.asObservable());

      component.onUpload();
      expect(component.uploadInProgress).toBeTrue();
      expect(mockVideoService.uploadVideo).toHaveBeenCalledWith(
        mockFile,
        'Test Title',
        'Test Desc',
        ['tag1', 'tag2']
      );

      // Simulate Sent event
      const sentEvent: HttpSentEvent = { type: HttpEventType.Sent };
      uploadEvents.next(sentEvent);
      tick(); fixture.detectChanges();
      // No specific UI change for Sent event usually, but progress starts

      // Simulate UploadProgress event
      const progressEvent: HttpProgressEvent = { type: HttpEventType.UploadProgress, loaded: 50, total: 100 };
      uploadEvents.next(progressEvent);
      tick(); fixture.detectChanges();
      expect(component.uploadProgress).toBe(50);
      
      const progressBar = fixture.debugElement.query(By.css('.progress-bar'));
      expect(progressBar).toBeTruthy();
      expect(progressBar.nativeElement.style.width).toBe('50%');
      expect(progressBar.nativeElement.textContent.trim()).toBe('50%');


      // Simulate another UploadProgress event
      const progressEvent2: HttpProgressEvent = { type: HttpEventType.UploadProgress, loaded: 100, total: 100 };
      uploadEvents.next(progressEvent2);
      tick(); fixture.detectChanges();
      expect(component.uploadProgress).toBe(100);


      // Simulate Response event (successful upload)
      const responseEvent = new HttpResponse({ body: { message: 'Upload successful' }, status: 200 });
      uploadEvents.next(responseEvent);
      tick(); fixture.detectChanges();

      expect(component.successMessage).toBe('Video uploaded successfully!');
      expect(component.uploadInProgress).toBeFalse();
      expect(component.selectedFile).toBeNull(); // Form should reset
      
      uploadEvents.complete(); // Complete the observable
    }));

    it('should handle error from VideoService.uploadVideo', fakeAsync(() => {
      const mockFile = new File(['content'], 'video.mp4');
      component.selectedFile = mockFile;
      mockVideoService.uploadVideo.and.returnValue(throwError(() => new Error('Upload failed miserably')));

      component.onUpload();
      tick();
      fixture.detectChanges();

      expect(component.errorMessage).toBe('Upload failed: Upload failed miserably');
      expect(component.uploadInProgress).toBeFalse();
      expect(component.uploadProgress).toBe(0);
    }));

    it('should reset form on successful upload', fakeAsync(() => {
        const mockFile = new File(['content'], 'video.mp4', { type: 'video/mp4' });
        component.selectedFile = mockFile;
        component.videoTitle = 'Old Title';
        component.videoDescription = 'Old Desc';
        component.videoTags = 'oldtag';

        const uploadEvents = new Subject<any>();
        mockVideoService.uploadVideo.and.returnValue(uploadEvents.asObservable());
        
        component.onUpload();
        tick();

        const responseEvent = new HttpResponse({ body: { message: 'Uploaded' } });
        uploadEvents.next(responseEvent);
        tick();
        fixture.detectChanges();

        expect(component.successMessage).toBe('Video uploaded successfully!');
        expect(component.selectedFile).toBeNull();
        expect(component.videoTitle).toBe('');
        expect(component.videoDescription).toBe('');
        expect(component.videoTags).toBe('');
        // Check file input reset (visual check is harder, but model should be null)
        const fileInput = fixture.debugElement.query(By.css('#videoFile')).nativeElement as HTMLInputElement;
        expect(fileInput.value).toBe(''); // This is what component.resetForm() does
    }));
  });
});
