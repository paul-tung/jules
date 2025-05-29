import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { of, throwError } from 'rxjs';
import { By } from '@angular/platform-browser';

import { VideoListComponent } from './video-list';
import { VideoService, Video, PaginatedVideosResponse } from '../../services/video';
import { FolderService, Folder } from '../../services/folder';

describe('VideoListComponent', () => {
  let component: VideoListComponent;
  let fixture: ComponentFixture<VideoListComponent>;
  let mockVideoService: jasmine.SpyObj<VideoService>;
  let mockFolderService: jasmine.SpyObj<FolderService>;
  let router: Router;

  const mockVideos: Video[] = [
    { ID: 'vid1', title: 'Video A', folderId: 'folder1', userId: 'u1', originalFilename:'', storagePath:'', status:'', uploadedAt:'', updatedAt:'' },
    { ID: 'vid2', title: 'Video B (Unassigned)', folderId: null, userId: 'u1', originalFilename:'', storagePath:'', status:'', uploadedAt:'', updatedAt:'' },
    { ID: 'vid3', title: 'Video C', folderId: 'folder1', userId: 'u1', originalFilename:'', storagePath:'', status:'', uploadedAt:'', updatedAt:'' },
  ];
  const mockPaginatedResponse: PaginatedVideosResponse = {
    items: mockVideos, currentPage: 1, pageSize: 10, totalItems: 3, totalPages: 1
  };
  const mockFolders: Folder[] = [
    { ID: 'folder1', name: 'Folder One', userId: 'u1', createdAt: '', updatedAt: '' },
    { ID: 'folder2', name: 'Folder Two', userId: 'u1', createdAt: '', updatedAt: '' },
  ];

  beforeEach(async () => {
    mockVideoService = jasmine.createSpyObj('VideoService', ['getVideos', 'deleteVideo']);
    mockFolderService = jasmine.createSpyObj('FolderService', ['getFolders', 'createFolder', 'updateFolder', 'deleteFolder']);

    // Setup default return values
    mockVideoService.getVideos.and.returnValue(of(mockPaginatedResponse));
    mockFolderService.getFolders.and.returnValue(of(mockFolders));
    mockFolderService.createFolder.and.callFake((name: string) => 
        of({ ID: 'newFolderId', name: name, userId: 'u1', createdAt: '', updatedAt: '' })
    );
    mockFolderService.updateFolder.and.callFake((id: string, name: string) =>
        of({ ID: id, name: name, userId: 'u1', createdAt: '', updatedAt: '', parentFolderId: null })
    );
    mockFolderService.deleteFolder.and.returnValue(of({ message: 'Folder deleted' }));
    mockVideoService.deleteVideo.and.returnValue(of({ message: 'Video deleted' }));


    await TestBed.configureTestingModule({
      imports: [
        CommonModule,
        FormsModule,
        RouterTestingModule,
        VideoListComponent // Standalone
      ],
      providers: [
        { provide: VideoService, useValue: mockVideoService },
        { provide: FolderService, useValue: mockFolderService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(VideoListComponent);
    component = fixture.componentInstance;
    router = TestBed.inject(Router);
    spyOn(router, 'navigate'); // Spy on router navigation
    fixture.detectChanges(); // ngOnInit will be called
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should load folders and videos on init', fakeAsync(() => {
    tick(); // Allow async operations in ngOnInit to complete
    expect(mockFolderService.getFolders).toHaveBeenCalledWith('root');
    expect(component.folders.length).toBe(2);
    expect(mockVideoService.getVideos).toHaveBeenCalledWith(1, component.pageSize);
    // Default client-side filtering: all videos are shown initially
    expect(component.videos.length).toBe(3); 
  }));

  describe('Folder Operations', () => {
    it('should create a new folder', fakeAsync(() => {
      component.newFolderName = '  New Test Folder  ';
      component.createFolder();
      tick();
      fixture.detectChanges();

      expect(mockFolderService.createFolder).toHaveBeenCalledWith('New Test Folder');
      expect(component.folders.length).toBe(3); // 2 initial + 1 new
      expect(component.folders.find(f => f.name === 'New Test Folder')).toBeTruthy();
      expect(component.newFolderName).toBe(''); // Should reset
    }));

    it('should not create folder if name is empty', () => {
        component.newFolderName = '   ';
        component.createFolder();
        expect(mockFolderService.createFolder).not.toHaveBeenCalled();
        expect(component.folderError).toBe('Folder name cannot be empty.');
    });

    it('should rename a folder', fakeAsync(() => {
      const folderToRename = mockFolders[0]; // Folder One
      component.startRenameFolder(folderToRename);
      component.renamingFolderName = 'Updated Folder One';
      component.submitRenameFolder();
      tick();
      fixture.detectChanges();

      expect(mockFolderService.updateFolder).toHaveBeenCalledWith(folderToRename.ID, 'Updated Folder One');
      expect(component.folders.find(f => f.ID === folderToRename.ID)?.name).toBe('Updated Folder One');
      expect(component.renamingFolder).toBeNull();
    }));
    
    it('should cancel renaming a folder', () => {
        component.startRenameFolder(mockFolders[0]);
        expect(component.renamingFolder).toEqual(mockFolders[0]);
        component.cancelRenameFolder();
        expect(component.renamingFolder).toBeNull();
    });


    it('should delete a folder after confirmation', fakeAsync(() => {
      spyOn(window, 'confirm').and.returnValue(true);
      const folderToDelete = mockFolders[0];
      component.confirmDeleteFolder(folderToDelete);
      tick();
      fixture.detectChanges();

      expect(window.confirm).toHaveBeenCalledWith(`Are you sure you want to delete the folder "${folderToDelete.name}"? This action cannot be undone. Ensure the folder is empty.`);
      expect(mockFolderService.deleteFolder).toHaveBeenCalledWith(folderToDelete.ID);
      expect(component.folders.length).toBe(1); // 2 initial - 1 deleted
      expect(component.folders.find(f => f.ID === folderToDelete.ID)).toBeFalsy();
    }));

     it('should not delete a folder if confirmation is cancelled', () => {
        spyOn(window, 'confirm').and.returnValue(false);
        component.confirmDeleteFolder(mockFolders[0]);
        expect(mockFolderService.deleteFolder).not.toHaveBeenCalled();
    });
  });

  describe('Video Filtering and Listing', () => {
    beforeEach(fakeAsync(() => {
        // Reset to default state for each filtering test
        component.selectedFolderId = null;
        component.isFilteringUnassigned = false;
        mockVideoService.getVideos.and.returnValue(of({...mockPaginatedResponse, items: [...mockVideos]})); // Fresh copy
        component.loadInitialData();
        tick();
        fixture.detectChanges();
    }));

    it('should filter videos by selected folder ID (client-side)', fakeAsync(() => {
      component.filterByFolder('folder1');
      tick(); // for loadVideos call
      fixture.detectChanges();
      
      expect(component.selectedFolderId).toBe('folder1');
      expect(component.isFilteringUnassigned).toBeFalse();
      expect(component.videos.length).toBe(2); // Video A, Video C
      expect(component.videos.every(v => v.folderId === 'folder1')).toBeTrue();
      expect(component.currentViewTitle).toBe('Folder One Videos');
    }));

    it('should filter for unassigned videos (client-side)', fakeAsync(() => {
      component.filterByFolder('unassigned');
      tick();
      fixture.detectChanges();

      expect(component.selectedFolderId).toBeNull();
      expect(component.isFilteringUnassigned).toBeTrue();
      expect(component.videos.length).toBe(1); // Video B
      expect(component.videos[0].ID).toBe('vid2');
      expect(component.currentViewTitle).toBe('Unassigned Videos');
    }));

    it('should show all videos when "All Videos" is selected (client-side)', fakeAsync(() => {
      // First, filter by a folder
      component.filterByFolder('folder1');
      tick(); fixture.detectChanges();
      expect(component.videos.length).toBe(2);

      // Then, select "All Videos"
      component.filterByFolder(null);
      tick(); fixture.detectChanges();
      
      expect(component.selectedFolderId).toBeNull();
      expect(component.isFilteringUnassigned).toBeFalse();
      expect(component.videos.length).toBe(3); // All videos
      expect(component.currentViewTitle).toBe('All Videos');
    }));
    
    it('getFolderName should return correct name or N/A', () => {
        expect(component.getFolderName('folder1')).toBe('Folder One');
        expect(component.getFolderName('nonexistent')).toBe('Unknown Folder');
        expect(component.getFolderName(null)).toBe('N/A');
        expect(component.getFolderName(undefined)).toBe('N/A');
    });
  });

  describe('Video Actions', () => {
    it('should navigate to edit video page', () => {
      component.editVideo('vid1');
      expect(router.navigate).toHaveBeenCalledWith(['/videos', 'vid1', 'edit']);
    });

    it('should delete a video after confirmation', fakeAsync(() => {
      spyOn(window, 'confirm').and.returnValue(true);
      mockVideoService.getVideos.calls.reset(); // Reset because deleteVideoAction calls loadVideos

      component.confirmDelete('vid1', 'Video A');
      tick();
      
      expect(window.confirm).toHaveBeenCalledWith('Are you sure you want to delete the video "Video A"?');
      expect(mockVideoService.deleteVideo).toHaveBeenCalledWith('vid1');
      tick(); // for deleteVideo observable
      expect(mockVideoService.getVideos).toHaveBeenCalled(); // loadVideos is called
    }));
  });
  
  describe('Pagination', () => {
    it('goToPage should call loadVideos with the new page number', () => {
        component.totalPages = 5;
        component.currentPage = 2;
        spyOn(component, 'loadVideos'); // Spy on the component's own method

        component.goToPage(3);
        expect(component.loadVideos).toHaveBeenCalledWith(3);
    });

    it('goToPage should not call loadVideos if page is invalid or same as current', () => {
        component.totalPages = 3;
        component.currentPage = 1;
        spyOn(component, 'loadVideos');

        component.goToPage(0); // Invalid
        component.goToPage(4); // Invalid (beyond totalPages)
        component.goToPage(1); // Same as current
        
        expect(component.loadVideos).not.toHaveBeenCalled();
    });
  });

});
