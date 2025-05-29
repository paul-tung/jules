import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { of, throwError, Subject } from 'rxjs';
import { By } from '@angular/platform-browser';
import { DomSanitizer } from '@angular/platform-browser';


import { VideoEditComponent } from './video-edit';
import { VideoService, Video, UpdateVideoPayload } from '../../services/video';
import { FolderService, Folder } from '../../services/folder';

describe('VideoEditComponent', () => {
  let component: VideoEditComponent;
  let fixture: ComponentFixture<VideoEditComponent>;
  let mockVideoService: jasmine.SpyObj<VideoService>;
  let mockFolderService: jasmine.SpyObj<FolderService>;
  let mockActivatedRoute: any; // Partial mock for ActivatedRoute
  let router: Router;
  let sanitizer: DomSanitizer;


  const videoId = 'vid123';
  const mockVideo: Video = {
    ID: videoId, title: 'Test Video', description: 'Desc', tags: ['a', 'b'], status: 'active',
    folderId: 'folder1', userId: 'u1', originalFilename: '', storagePath: './uploads/fake.mp4', 
    uploadedAt: new Date().toISOString(), updatedAt: new Date().toISOString()
  };
  const mockFolders: Folder[] = [
    { ID: 'folder1', name: 'Folder One', userId: 'u1', createdAt: '', updatedAt: '' },
    { ID: 'folder2', name: 'Folder Two', userId: 'u1', createdAt: '', updatedAt: '' },
  ];

  beforeEach(async () => {
    mockVideoService = jasmine.createSpyObj('VideoService', ['getVideo', 'updateVideo', 'assignVideoToFolder']);
    mockFolderService = jasmine.createSpyObj('FolderService', ['getFolders']);

    mockActivatedRoute = {
      paramMap: of(new Map([['videoId', videoId]])) // Simulate route parameter
      // paramMap: new BehaviorSubject(convertToParamMap({ videoId: videoId })) // Alternative for BehaviorSubject
    };

    mockVideoService.getVideo.and.returnValue(of(mockVideo));
    mockFolderService.getFolders.and.returnValue(of(mockFolders));

    await TestBed.configureTestingModule({
      imports: [
        CommonModule,
        FormsModule,
        RouterTestingModule,
        VideoEditComponent // Standalone
      ],
      providers: [
        { provide: VideoService, useValue: mockVideoService },
        { provide: FolderService, useValue: mockFolderService },
        { provide: ActivatedRoute, useValue: mockActivatedRoute }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(VideoEditComponent);
    component = fixture.componentInstance;
    router = TestBed.inject(Router);
    sanitizer = TestBed.inject(DomSanitizer);
    spyOn(router, 'navigate'); // Spy on router navigation
    fixture.detectChanges(); // ngOnInit will be called
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should load video details and folders on init', fakeAsync(() => {
    tick(); // for observables in ngOnInit
    fixture.detectChanges();

    expect(mockVideoService.getVideo).toHaveBeenCalledWith(videoId);
    expect(component.video).toEqual(mockVideo);
    expect(component.editableVideo.title).toBe(mockVideo.title);
    expect(component.tagsString).toBe('a, b');
    expect(component.selectedFolderId).toBe('folder1');

    expect(mockFolderService.getFolders).toHaveBeenCalledWith('root');
    expect(component.availableFolders.length).toBe(2);
    
    // Check video player URL (basic check, actual sanitization is complex)
    // As storagePath is './uploads/fake.mp4', it will be set directly
    expect(component.videoPlayerUrl).toBe(mockVideo.storagePath);
  }));

  describe('Save Metadata Changes', () => {
    beforeEach(fakeAsync(() => {
      tick(); // Ensure ngOnInit completes
      fixture.detectChanges();
    }));

    it('should call VideoService.updateVideo with correct payload', fakeAsync(() => {
      const updatedTitle = 'Updated Test Video';
      const updatedDesc = 'Updated Desc';
      const updatedTagsStr = 'c, d, e';
      const updatedStatus = 'private';

      component.editableVideo.title = updatedTitle;
      component.editableVideo.description = updatedDesc;
      component.tagsString = updatedTagsStr;
      component.editableVideo.status = updatedStatus;
      
      const expectedPayload: UpdateVideoPayload = {
        title: updatedTitle,
        description: updatedDesc,
        tags: ['c', 'd', 'e'],
        status: updatedStatus
      };
      mockVideoService.updateVideo.and.returnValue(of({ ...mockVideo, ...expectedPayload }));

      component.saveChanges();
      tick(); // for async updateVideo call
      fixture.detectChanges();

      expect(mockVideoService.updateVideo).toHaveBeenCalledWith(videoId, expectedPayload);
      expect(component.successMessage).toBe('Video metadata updated successfully!');
    }));

    it('should display error message if metadata update fails', fakeAsync(() => {
      mockVideoService.updateVideo.and.returnValue(throwError(() => new Error('Update failed')));
      component.saveChanges();
      tick();
      fixture.detectChanges();

      expect(component.updateErrorMessage).toBe('Failed to update video metadata: Update failed');
    }));
  });

  describe('Assign to Folder', () => {
     beforeEach(fakeAsync(() => {
      tick(); // Ensure ngOnInit completes
      fixture.detectChanges();
    }));

    it('should call VideoService.assignVideoToFolder with selected folder ID', fakeAsync(() => {
      const targetFolderId = 'folder2';
      component.selectedFolderId = targetFolderId;
      mockVideoService.assignVideoToFolder.and.returnValue(of({ ...mockVideo, folderId: targetFolderId }));

      component.assignToFolder();
      tick();
      fixture.detectChanges();

      expect(mockVideoService.assignVideoToFolder).toHaveBeenCalledWith(videoId, targetFolderId);
      expect(component.folderAssignmentMessage.text).toBe('Video folder updated successfully!');
      expect(component.folderAssignmentMessage.type).toBe('success');
      expect(component.video?.folderId).toBe(targetFolderId);
    }));

    it('should unassign video if selectedFolderId is null', fakeAsync(() => {
      component.selectedFolderId = null; // User chooses "Unassigned"
      mockVideoService.assignVideoToFolder.and.returnValue(of({ ...mockVideo, folderId: null }));
      
      component.assignToFolder();
      tick();
      fixture.detectChanges();

      expect(mockVideoService.assignVideoToFolder).toHaveBeenCalledWith(videoId, null);
      expect(component.folderAssignmentMessage.text).toBe('Video folder updated successfully!');
      expect(component.video?.folderId).toBeNull();
    }));
    
    it('should handle error during folder assignment', fakeAsync(() => {
      component.selectedFolderId = 'folder1';
      mockVideoService.assignVideoToFolder.and.returnValue(throwError(() => new Error('Assign error')));
      
      component.assignToFolder();
      tick();
      fixture.detectChanges();

      expect(component.folderAssignmentMessage.text).toBe('Failed to update video folder: Assign error');
      expect(component.folderAssignmentMessage.type).toBe('error');
    }));
  });

  it('goBackToList should navigate to /videos', () => {
    component.goBackToList();
    expect(router.navigate).toHaveBeenCalledWith(['/videos']);
  });
  
  it('should unsubscribe from route params on destroy', fakeAsync(() => {
    tick(); // ngOnInit
    spyOn(component['routeSub']!, 'unsubscribe'); // Access private member for test
    
    component.ngOnDestroy();
    expect(component['routeSub']!.unsubscribe).toHaveBeenCalled();
  }));

});
